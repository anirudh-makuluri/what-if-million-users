package com.example.leaderboard.service;

public class PlayerNotFoundException extends RuntimeException {

    public PlayerNotFoundException(String playerId) {
        super("Player %s was not found".formatted(playerId));
    }
}
