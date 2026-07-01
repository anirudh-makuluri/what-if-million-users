package com.example.leaderboard.web;

import java.time.Instant;

public record ScoreSubmissionResponse(
        String playerId,
        String displayName,
        long score,
        long rank,
        boolean scoreAccepted,
        Instant createdAt,
        Instant updatedAt
) {
}
