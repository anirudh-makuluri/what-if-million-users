package com.example.leaderboard.service;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

import com.example.leaderboard.persistence.LeaderboardEntryEntity;

import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.stereotype.Component;

@Component
public class RedisLeaderboardStore {

    private static final String SCORES_KEY = "leaderboard:scores";
    private static final String PLAYER_STATE_PREFIX = "leaderboard:player:";

    private final StringRedisTemplate redisTemplate;

    public RedisLeaderboardStore(StringRedisTemplate redisTemplate) {
        this.redisTemplate = redisTemplate;
    }

    public void upsert(LeaderboardEntryEntity entry) {
        redisTemplate.executePipelined((org.springframework.data.redis.core.RedisCallback<Object>) connection -> {
            byte[] scoresKey = redisTemplate.getStringSerializer().serialize(SCORES_KEY);
            byte[] playerId = redisTemplate.getStringSerializer().serialize(entry.getPlayerId());
            byte[] stateKey = redisTemplate.getStringSerializer().serialize(stateKey(entry.getPlayerId()));

            connection.zAdd(scoresKey, entry.getScore(), playerId);
            connection.hSet(stateKey, serialize("playerId"), serialize(entry.getPlayerId()));
            connection.hSet(stateKey, serialize("displayName"), serialize(entry.getDisplayName()));
            connection.hSet(stateKey, serialize("updatedAt"), serialize(String.valueOf(entry.getUpdatedAt().toEpochMilli())));
            return null;
        });
    }

    public void upsertAll(List<LeaderboardEntryEntity> entries) {
        if (entries.isEmpty()) {
            return;
        }

        redisTemplate.executePipelined((org.springframework.data.redis.core.RedisCallback<Object>) connection -> {
            byte[] scoresKey = redisTemplate.getStringSerializer().serialize(SCORES_KEY);
            for (LeaderboardEntryEntity entry : entries) {
                byte[] playerId = redisTemplate.getStringSerializer().serialize(entry.getPlayerId());
                byte[] stateKey = redisTemplate.getStringSerializer().serialize(stateKey(entry.getPlayerId()));

                connection.zAdd(scoresKey, entry.getScore(), playerId);
                connection.hSet(stateKey, serialize("playerId"), serialize(entry.getPlayerId()));
                connection.hSet(stateKey, serialize("displayName"), serialize(entry.getDisplayName()));
                connection.hSet(stateKey, serialize("updatedAt"), serialize(String.valueOf(entry.getUpdatedAt().toEpochMilli())));
            }
            return null;
        });
    }

    public Long rankOf(String playerId) {
        Long rank = redisTemplate.opsForZSet().reverseRank(SCORES_KEY, playerId);
        return rank == null ? null : rank + 1;
    }

    public Long size() {
        return redisTemplate.opsForZSet().size(SCORES_KEY);
    }

    public List<RankedScore> top(int limit) {
        if (limit <= 0) {
            return List.of();
        }

        Set<ZSetOperations.TypedTuple<String>> tuples =
                redisTemplate.opsForZSet().reverseRangeWithScores(SCORES_KEY, 0, limit - 1L);
        return toRankedScores(tuples, 1);
    }

    public List<RankedScore> around(String playerId, int before, int after) {
        Long zeroBasedRank = redisTemplate.opsForZSet().reverseRank(SCORES_KEY, playerId);
        if (zeroBasedRank == null) {
            return List.of();
        }

        long start = Math.max(zeroBasedRank - before, 0);
        long end = zeroBasedRank + after;
        Set<ZSetOperations.TypedTuple<String>> tuples =
                redisTemplate.opsForZSet().reverseRangeWithScores(SCORES_KEY, start, end);
        return toRankedScores(tuples, start + 1);
    }

    private List<RankedScore> toRankedScores(Set<ZSetOperations.TypedTuple<String>> tuples, long startingRank) {
        if (tuples == null || tuples.isEmpty()) {
            return List.of();
        }

        List<RankedScore> scores = new ArrayList<>(tuples.size());
        long rank = startingRank;
        for (ZSetOperations.TypedTuple<String> tuple : tuples) {
            if (tuple.getValue() == null || tuple.getScore() == null) {
                continue;
            }
            scores.add(new RankedScore(tuple.getValue(), tuple.getScore().longValue(), rank));
            rank++;
        }
        return scores;
    }

    private String stateKey(String playerId) {
        return PLAYER_STATE_PREFIX + playerId;
    }

    private byte[] serialize(String value) {
        return redisTemplate.getStringSerializer().serialize(value);
    }

    public record RankedScore(String playerId, long score, long rank) {
    }
}
