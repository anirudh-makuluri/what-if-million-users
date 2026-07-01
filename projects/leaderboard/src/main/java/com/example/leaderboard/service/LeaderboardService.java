package com.example.leaderboard.service;

import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import com.example.leaderboard.config.LeaderboardProperties;
import com.example.leaderboard.persistence.LeaderboardEntryEntity;
import com.example.leaderboard.persistence.LeaderboardEntryRepository;
import com.example.leaderboard.web.LeaderboardResponse;
import com.example.leaderboard.web.PlayerScoreResponse;
import com.example.leaderboard.web.RankedPlayerResponse;
import com.example.leaderboard.web.ScoreSubmissionRequest;
import com.example.leaderboard.web.ScoreSubmissionResponse;

import jakarta.persistence.EntityManager;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.orm.ObjectOptimisticLockingFailureException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class LeaderboardService {

    private static final Logger log = LoggerFactory.getLogger(LeaderboardService.class);
    private static final int MAX_LIMIT = 100;
    private static final int MAX_NEIGHBORS = 25;

    private final LeaderboardEntryRepository leaderboardEntryRepository;
    private final RedisLeaderboardStore redisLeaderboardStore;
    private final LeaderboardProperties properties;
    private final EntityManager entityManager;

    public LeaderboardService(
            LeaderboardEntryRepository leaderboardEntryRepository,
            RedisLeaderboardStore redisLeaderboardStore,
            LeaderboardProperties properties,
            EntityManager entityManager
    ) {
        this.leaderboardEntryRepository = leaderboardEntryRepository;
        this.redisLeaderboardStore = redisLeaderboardStore;
        this.properties = properties;
        this.entityManager = entityManager;
    }

    @Transactional
    public ScoreSubmissionResponse submitScore(ScoreSubmissionRequest request) {
        String playerId = request.playerId().trim();
        String displayName = request.displayName().trim();
        long submittedScore = request.score();

        for (int attempt = 1; attempt <= properties.getMaxWriteRetries(); attempt++) {
            try {
                LeaderboardEntryEntity entry;
                boolean scoreAccepted;

                LeaderboardEntryEntity existing = leaderboardEntryRepository.findById(playerId).orElse(null);
                if (existing == null) {
                    entry = new LeaderboardEntryEntity();
                    entry.setPlayerId(playerId);
                    entry.setDisplayName(displayName);
                    entry.setScore(submittedScore);
                    scoreAccepted = true;
                } else {
                    entry = existing;
                    entry.setDisplayName(displayName);
                    scoreAccepted = submittedScore >= existing.getScore();
                    if (submittedScore > existing.getScore()) {
                        entry.setScore(submittedScore);
                    }
                }

                LeaderboardEntryEntity saved = leaderboardEntryRepository.saveAndFlush(entry);
                safeCacheSync(saved, "submit-score");
                long rank = resolveRank(saved);
                return new ScoreSubmissionResponse(
                        saved.getPlayerId(),
                        saved.getDisplayName(),
                        saved.getScore(),
                        rank,
                        scoreAccepted,
                        saved.getCreatedAt(),
                        saved.getUpdatedAt()
                );
            } catch (ObjectOptimisticLockingFailureException | DataIntegrityViolationException ex) {
                if (attempt == properties.getMaxWriteRetries()) {
                    throw new LeaderboardWriteConflictException(playerId);
                }
                log.debug("Retrying concurrent score write for player {} on attempt {}", playerId, attempt, ex);
            }
        }

        throw new LeaderboardWriteConflictException(playerId);
    }

    @Transactional(readOnly = true)
    public PlayerScoreResponse getPlayer(String playerId) {
        LeaderboardEntryEntity entry = loadPlayer(playerId);
        return toPlayerResponse(entry, resolveRank(entry));
    }

    @Transactional(readOnly = true)
    public LeaderboardResponse getTop(int limit) {
        int normalizedLimit = normalizeLimit(limit);
        try {
            List<RedisLeaderboardStore.RankedScore> rankedScores = redisLeaderboardStore.top(normalizedLimit);
            if (!rankedScores.isEmpty()) {
                return new LeaderboardResponse(toRankedResponses(rankedScores));
            }
        } catch (DataAccessException ex) {
            log.warn("Redis top lookup failed, falling back to MSSQL", ex);
        }

        return topFromDatabase(normalizedLimit);
    }

    @Transactional(readOnly = true)
    public LeaderboardResponse getNeighbors(String playerId, int before, int after) {
        loadPlayer(playerId);

        int normalizedBefore = normalizeNeighborCount(before);
        int normalizedAfter = normalizeNeighborCount(after);

        try {
            List<RedisLeaderboardStore.RankedScore> rankedScores =
                    redisLeaderboardStore.around(playerId, normalizedBefore, normalizedAfter);
            if (!rankedScores.isEmpty()) {
                return new LeaderboardResponse(toRankedResponses(rankedScores));
            }
        } catch (DataAccessException ex) {
            log.warn("Redis around-player lookup failed, falling back to MSSQL", ex);
        }

        return neighborsFromDatabase(playerId, normalizedBefore, normalizedAfter);
    }

    @Transactional(readOnly = true)
    public int hydrateLeaderboard() {
        int batchSize = properties.getHydrationBatchSize();
        int hydrated = 0;
        int pageNumber = 0;

        while (true) {
            Page<LeaderboardEntryEntity> page =
                    leaderboardEntryRepository.findAllByOrderByScoreDescPlayerIdAsc(PageRequest.of(pageNumber, batchSize));
            if (page.isEmpty()) {
                break;
            }

            try {
                redisLeaderboardStore.upsertAll(page.getContent());
            } catch (DataAccessException ex) {
                log.warn("Redis hydration failed after {} entries. MSSQL remains the source of truth.", hydrated, ex);
                break;
            }

            hydrated += page.getNumberOfElements();
            if (!page.hasNext()) {
                break;
            }
            pageNumber++;
        }

        return hydrated;
    }

    @Transactional(readOnly = true)
    public int hydrateLeaderboardIfCacheBehind() {
        try {
            Long redisSize = redisLeaderboardStore.size();
            long databaseCount = leaderboardEntryRepository.count();
            if (redisSize != null && redisSize >= databaseCount) {
                return 0;
            }
        } catch (DataAccessException ex) {
            log.warn("Unable to compare Redis leaderboard size during cache audit", ex);
            return 0;
        }

        return hydrateLeaderboard();
    }

    private PlayerScoreResponse toPlayerResponse(LeaderboardEntryEntity entry, long rank) {
        return new PlayerScoreResponse(
                entry.getPlayerId(),
                entry.getDisplayName(),
                entry.getScore(),
                rank,
                entry.getCreatedAt(),
                entry.getUpdatedAt()
        );
    }

    private LeaderboardResponse topFromDatabase(int limit) {
        List<LeaderboardEntryEntity> window = loadOrderedWindow(0, limit);
        return new LeaderboardResponse(toRankedResponses(window, 1));
    }

    private LeaderboardResponse neighborsFromDatabase(String playerId, int before, int after) {
        LeaderboardEntryEntity player = loadPlayer(playerId);
        long rank = computeRankFromDatabase(player);
        long startIndex = Math.max(rank - 1 - before, 0);
        int windowSize = before + after + 1;
        List<LeaderboardEntryEntity> window = loadOrderedWindow(startIndex, windowSize);
        return new LeaderboardResponse(toRankedResponses(window, startIndex + 1));
    }

    private List<RankedPlayerResponse> toRankedResponses(List<RedisLeaderboardStore.RankedScore> rankedScores) {
        Map<String, LeaderboardEntryEntity> entriesById = leaderboardEntryRepository.findAllById(
                        rankedScores.stream().map(RedisLeaderboardStore.RankedScore::playerId).toList())
                .stream()
                .sorted(Comparator.comparing(LeaderboardEntryEntity::getPlayerId))
                .collect(java.util.stream.Collectors.toMap(
                        LeaderboardEntryEntity::getPlayerId,
                        entry -> entry,
                        (left, right) -> left,
                        LinkedHashMap::new
                ));

        return rankedScores.stream()
                .map(score -> {
                    LeaderboardEntryEntity entry = entriesById.get(score.playerId());
                    if (entry == null) {
                        return new RankedPlayerResponse(score.rank(), score.playerId(), score.playerId(), score.score(), null);
                    }
                    return new RankedPlayerResponse(
                            score.rank(),
                            entry.getPlayerId(),
                            entry.getDisplayName(),
                            entry.getScore(),
                            entry.getUpdatedAt()
                    );
                })
                .toList();
    }

    private List<RankedPlayerResponse> toRankedResponses(List<LeaderboardEntryEntity> entries, long startingRank) {
        long rank = startingRank;
        List<RankedPlayerResponse> rankedPlayers = new java.util.ArrayList<>(entries.size());
        for (LeaderboardEntryEntity entry : entries) {
            rankedPlayers.add(new RankedPlayerResponse(
                    rank,
                    entry.getPlayerId(),
                    entry.getDisplayName(),
                    entry.getScore(),
                    entry.getUpdatedAt()
            ));
            rank++;
        }
        return rankedPlayers;
    }

    private LeaderboardEntryEntity loadPlayer(String playerId) {
        return leaderboardEntryRepository.findById(playerId).orElseThrow(() -> new PlayerNotFoundException(playerId));
    }

    private long resolveRank(LeaderboardEntryEntity entry) {
        try {
            Long rank = redisLeaderboardStore.rankOf(entry.getPlayerId());
            if (rank != null) {
                return rank;
            }
        } catch (DataAccessException ex) {
            log.warn("Redis rank lookup failed for player {}, falling back to MSSQL", entry.getPlayerId(), ex);
        }

        return computeRankFromDatabase(entry);
    }

    private long computeRankFromDatabase(LeaderboardEntryEntity entry) {
        return leaderboardEntryRepository.countByScoreGreaterThan(entry.getScore())
                + leaderboardEntryRepository.countByScoreAndPlayerIdLessThan(entry.getScore(), entry.getPlayerId())
                + 1;
    }

    private List<LeaderboardEntryEntity> loadOrderedWindow(long startIndex, int size) {
        if (size <= 0) {
            return List.of();
        }

        return entityManager.createQuery(
                        "select e from LeaderboardEntryEntity e order by e.score desc, e.playerId asc",
                        LeaderboardEntryEntity.class
                )
                .setFirstResult(Math.toIntExact(startIndex))
                .setMaxResults(size)
                .getResultList();
    }

    private void safeCacheSync(LeaderboardEntryEntity entry, String operation) {
        try {
            redisLeaderboardStore.upsert(entry);
        } catch (DataAccessException ex) {
            log.warn(
                    "Redis sync failed during {} for player {}. MSSQL remains durable.",
                    operation,
                    entry.getPlayerId(),
                    ex
            );
        }
    }

    private int normalizeLimit(int limit) {
        if (limit <= 0) {
            return 10;
        }
        return Math.min(limit, MAX_LIMIT);
    }

    private int normalizeNeighborCount(int value) {
        if (value < 0) {
            return 0;
        }
        return Math.min(value, MAX_NEIGHBORS);
    }
}
