package com.example.leaderboard.web;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;

public record ScoreSubmissionRequest(
        @NotBlank(message = "playerId is required")
        String playerId,
        @NotBlank(message = "displayName is required")
        String displayName,
        @Min(value = 0, message = "score must be zero or greater")
        long score
) {
}
