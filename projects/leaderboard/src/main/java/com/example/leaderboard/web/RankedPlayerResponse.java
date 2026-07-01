package com.example.leaderboard.web;

import java.time.Instant;

public record RankedPlayerResponse(
        long rank,
        String playerId,
        String displayName,
        long score,
        Instant updatedAt
) {
}
