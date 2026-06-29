package com.example.jobqueue.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class JobMaintenanceScheduler {

    private static final Logger log = LoggerFactory.getLogger(JobMaintenanceScheduler.class);

    private final JobService jobService;

    public JobMaintenanceScheduler(JobService jobService) {
        this.jobService = jobService;
    }

    @EventListener(ApplicationReadyEvent.class)
    public void onReady() {
        int hydrated = jobService.hydrateQueuedJobs();
        log.info("Hydrated {} queued jobs into Redis after startup", hydrated);
    }

    @Scheduled(
            initialDelayString = "${job-queue.recovery-interval-ms:10000}",
            fixedDelayString = "${job-queue.recovery-interval-ms:10000}"
    )
    public void recoverExpiredJobs() {
        int recovered = jobService.recoverExpiredJobs();
        if (recovered > 0) {
            log.info("Recovered {} expired jobs back into the queue", recovered);
        }
    }

    @Scheduled(
            initialDelayString = "${job-queue.hydration-interval-ms:15000}",
            fixedDelayString = "${job-queue.hydration-interval-ms:15000}"
    )
    public void hydrateQueuedJobs() {
        int hydrated = jobService.hydrateQueuedJobs();
        if (hydrated > 0) {
            log.debug("Hydrated {} queued jobs into Redis", hydrated);
        }
    }
}
