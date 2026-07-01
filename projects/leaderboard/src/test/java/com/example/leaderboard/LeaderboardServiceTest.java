package com.example.leaderboard;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.time.Instant;
import java.util.Optional;

import com.example.leaderboard.config.LeaderboardProperties;
import com.example.leaderboard.persistence.LeaderboardEntryEntity;
import com.example.leaderboard.persistence.LeaderboardEntryRepository;
import com.example.leaderboard.service.LeaderboardService;
import com.example.leaderboard.service.RedisLeaderboardStore;
import com.example.leaderboard.web.PlayerScoreResponse;
import com.example.leaderboard.web.ScoreSubmissionRequest;
import com.example.leaderboard.web.ScoreSubmissionResponse;

import jakarta.persistence.EntityManager;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.dao.DataAccessResourceFailureException;

@ExtendWith(MockitoExtension.class)
class LeaderboardServiceTest {

    @Mock
    private LeaderboardEntryRepository repository;

    @Mock
    private RedisLeaderboardStore redisLeaderboardStore;

    @Mock
    private EntityManager entityManager;

    private LeaderboardService leaderboardService;

    @BeforeEach
    void setUp() {
        LeaderboardProperties properties = new LeaderboardProperties();
        properties.setMaxWriteRetries(3);
        leaderboardService = new LeaderboardService(repository, redisLeaderboardStore, properties, entityManager);
    }

    @Test
    void shouldKeepExistingHighScoreWhenLowerScoreIsSubmitted() {
        LeaderboardEntryEntity entry = player("player-1", "Nova", 1200, Instant.parse("2026-01-01T00:00:00Z"));

        when(repository.findById("player-1")).thenReturn(Optional.of(entry));
        when(repository.saveAndFlush(any(LeaderboardEntryEntity.class))).thenAnswer(invocation -> invocation.getArgument(0));
        when(redisLeaderboardStore.rankOf("player-1")).thenReturn(4L);

        ScoreSubmissionResponse response = leaderboardService.submitScore(
                new ScoreSubmissionRequest("player-1", "NovaPrime", 900)
        );

        assertThat(response.score()).isEqualTo(1200);
        assertThat(response.scoreAccepted()).isFalse();
        assertThat(response.displayName()).isEqualTo("NovaPrime");
        assertThat(response.rank()).isEqualTo(4L);
        verify(redisLeaderboardStore).upsert(any(LeaderboardEntryEntity.class));
    }

    @Test
    void shouldFallbackToDatabaseRankWhenRedisRankLookupFails() {
        LeaderboardEntryEntity entry = player("player-9", "Atlas", 4200, Instant.parse("2026-01-01T00:00:00Z"));

        when(repository.findById("player-9")).thenReturn(Optional.of(entry));
        when(redisLeaderboardStore.rankOf("player-9"))
                .thenThrow(new DataAccessResourceFailureException("redis unavailable"));
        when(repository.countByScoreGreaterThan(4200)).thenReturn(2L);
        when(repository.countByScoreAndPlayerIdLessThan(4200, "player-9")).thenReturn(1L);

        PlayerScoreResponse response = leaderboardService.getPlayer("player-9");

        assertThat(response.rank()).isEqualTo(4L);
        assertThat(response.score()).isEqualTo(4200);
    }

    private LeaderboardEntryEntity player(String playerId, String displayName, long score, Instant timestamp) {
        LeaderboardEntryEntity entry = new LeaderboardEntryEntity();
        entry.setPlayerId(playerId);
        entry.setDisplayName(displayName);
        entry.setScore(score);

        try {
            java.lang.reflect.Field createdAt = LeaderboardEntryEntity.class.getDeclaredField("createdAt");
            createdAt.setAccessible(true);
            createdAt.set(entry, timestamp);

            java.lang.reflect.Field updatedAt = LeaderboardEntryEntity.class.getDeclaredField("updatedAt");
            updatedAt.setAccessible(true);
            updatedAt.set(entry, timestamp);
        } catch (ReflectiveOperationException ex) {
            throw new IllegalStateException(ex);
        }

        return entry;
    }
}
