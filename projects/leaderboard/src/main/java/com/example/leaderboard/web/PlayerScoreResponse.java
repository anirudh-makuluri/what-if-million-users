package com.example.leaderboard.web;

import java.time.Instant;

public record PlayerScoreResponse(
        String playerId,
        String displayName,
        long score,
        long rank,
        Instant createdAt,
        Instant updatedAt
) {
}
