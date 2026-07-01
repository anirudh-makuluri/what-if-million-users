package com.example.leaderboard.web;

import java.time.Instant;
import java.util.List;

public record LeaderboardResponse(
        Instant generatedAt,
        int count,
        List<RankedPlayerResponse> entries
) {

    public LeaderboardResponse(List<RankedPlayerResponse> entries) {
        this(Instant.now(), entries.size(), entries);
    }
}
