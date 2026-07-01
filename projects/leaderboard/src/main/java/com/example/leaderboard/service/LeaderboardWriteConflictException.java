package com.example.leaderboard.service;

public class LeaderboardWriteConflictException extends RuntimeException {

    public LeaderboardWriteConflictException(String playerId) {
        super("Concurrent updates prevented a stable write for player %s".formatted(playerId));
    }
}
