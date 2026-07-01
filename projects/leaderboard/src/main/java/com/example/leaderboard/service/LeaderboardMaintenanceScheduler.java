package com.example.leaderboard.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class LeaderboardMaintenanceScheduler {

    private static final Logger log = LoggerFactory.getLogger(LeaderboardMaintenanceScheduler.class);

    private final LeaderboardService leaderboardService;

    public LeaderboardMaintenanceScheduler(LeaderboardService leaderboardService) {
        this.leaderboardService = leaderboardService;
    }

    @EventListener(ApplicationReadyEvent.class)
    public void onReady() {
        int hydrated = leaderboardService.hydrateLeaderboard();
        log.info("Hydrated {} leaderboard entries into Redis after startup", hydrated);
    }

    @Scheduled(
            initialDelayString = "${leaderboard.cache-audit-interval-ms:30000}",
            fixedDelayString = "${leaderboard.cache-audit-interval-ms:30000}"
    )
    public void auditCache() {
        int hydrated = leaderboardService.hydrateLeaderboardIfCacheBehind();
        if (hydrated > 0) {
            log.info("Rehydrated {} leaderboard entries after detecting cache drift", hydrated);
        }
    }
}
