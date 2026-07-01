package com.example.leaderboard.web;

import com.example.leaderboard.service.LeaderboardService;

import jakarta.validation.Valid;
import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;

import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@Validated
@RestController
@RequestMapping("/leaderboard")
public class LeaderboardController {

    private final LeaderboardService leaderboardService;

    public LeaderboardController(LeaderboardService leaderboardService) {
        this.leaderboardService = leaderboardService;
    }

    @PostMapping("/scores")
    public ScoreSubmissionResponse submitScore(@Valid @RequestBody ScoreSubmissionRequest request) {
        return leaderboardService.submitScore(request);
    }

    @GetMapping("/players/{playerId}")
    public PlayerScoreResponse getPlayer(@PathVariable String playerId) {
        return leaderboardService.getPlayer(playerId);
    }

    @GetMapping("/top")
    public LeaderboardResponse getTop(
            @RequestParam(defaultValue = "10") @Min(value = 1, message = "limit must be at least 1")
            @Max(value = 100, message = "limit must be at most 100") int limit
    ) {
        return leaderboardService.getTop(limit);
    }

    @GetMapping("/players/{playerId}/neighbors")
    public LeaderboardResponse getNeighbors(
            @PathVariable String playerId,
            @RequestParam(defaultValue = "2") @Min(value = 0, message = "before must be zero or greater")
            @Max(value = 25, message = "before must be at most 25") int before,
            @RequestParam(defaultValue = "2") @Min(value = 0, message = "after must be zero or greater")
            @Max(value = 25, message = "after must be at most 25") int after
    ) {
        return leaderboardService.getNeighbors(playerId, before, after);
    }
}
