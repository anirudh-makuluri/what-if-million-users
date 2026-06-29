package com.example.jobqueue.service;

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.UUID;

import com.example.jobqueue.config.JobQueueProperties;
import com.example.jobqueue.model.JobStatus;
import com.example.jobqueue.persistence.JobEntity;
import com.example.jobqueue.persistence.JobRepository;
import com.example.jobqueue.web.ClaimJobRequest;
import com.example.jobqueue.web.CompleteJobRequest;
import com.example.jobqueue.web.CreateJobRequest;
import com.example.jobqueue.web.FailJobRequest;
import com.example.jobqueue.web.JobResponse;

import jakarta.persistence.OptimisticLockException;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.data.domain.PageRequest;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class JobService {

    private static final Logger log = LoggerFactory.getLogger(JobService.class);

    private final JobRepository jobRepository;
    private final RedisQueueStore queueStore;
    private final RetryBackoffPolicy backoffPolicy;
    private final JobQueueProperties properties;

    public JobService(
            JobRepository jobRepository,
            RedisQueueStore queueStore,
            RetryBackoffPolicy backoffPolicy,
            JobQueueProperties properties
    ) {
        this.jobRepository = jobRepository;
        this.queueStore = queueStore;
        this.backoffPolicy = backoffPolicy;
        this.properties = properties;
    }

    @Transactional
    public JobResponse createJob(CreateJobRequest request) {
        Instant now = Instant.now();
        JobEntity job = new JobEntity();
        job.setId(UUID.randomUUID().toString());
        job.setPayload(request.payload().trim());
        job.setStatus(JobStatus.QUEUED);
        job.setRetryCount(0);
        job.setMaxRetries(request.maxRetries() == null ? properties.getDefaultMaxRetries() : request.maxRetries());
        job.setAvailableAt(now.plusSeconds(request.delaySeconds() == null ? 0L : request.delaySeconds()));

        JobEntity saved = jobRepository.saveAndFlush(job);
        safeQueueSync(saved, "enqueue");
        return toResponse(saved);
    }

    @Transactional(readOnly = true)
    public JobResponse getJob(String jobId) {
        return toResponse(loadJob(jobId));
    }

    @Transactional
    public JobResponse claimJob(String jobId, ClaimJobRequest request) {
        Instant now = Instant.now();
        JobEntity job = loadJob(jobId);
        assertQueuedAndAvailable(job, now);

        Duration lease = request.leaseSeconds() == null
                ? properties.getDefaultLease()
                : Duration.ofSeconds(request.leaseSeconds());
        Instant leaseUntil = now.plus(lease);

        boolean claimed = tryClaimWithRepair(job, now, leaseUntil);
        if (!claimed) {
            throw new JobConflictException("Job %s is already claimed or not ready".formatted(jobId));
        }

        Instant originalAvailableAt = job.getAvailableAt();
        try {
            job.setStatus(JobStatus.PROCESSING);
            job.setProcessorId(request.processorId().trim());
            job.setLockExpiresAt(leaseUntil);
            jobRepository.saveAndFlush(job);
            safeCacheSync(job, "claim");
            return toResponse(job);
        } catch (RuntimeException ex) {
            queueStore.restoreQueuedState(job.getId(), originalAvailableAt);
            throw ex;
        }
    }

    @Transactional
    public JobResponse completeJob(String jobId, CompleteJobRequest request) {
        JobEntity job = loadJob(jobId);
        assertOwnedProcessingJob(job, request.processorId());

        job.setStatus(JobStatus.COMPLETED);
        job.setProcessorId(request.processorId().trim());
        job.setLockExpiresAt(null);
        job.setCompletedAt(Instant.now());

        jobRepository.saveAndFlush(job);
        safeQueueSync(job, "complete");
        return toResponse(job);
    }

    @Transactional
    public JobResponse failJob(String jobId, FailJobRequest request) {
        JobEntity job = loadJob(jobId);
        assertOwnedProcessingJob(job, request.processorId());

        boolean retryable = request.retryable() == null || request.retryable();
        transitionFailure(job, request.errorMessage().trim(), retryable, Instant.now());

        jobRepository.saveAndFlush(job);
        safeQueueSync(job, "fail");
        return toResponse(job);
    }

    @Transactional
    public int recoverExpiredJobs() {
        Instant now = Instant.now();
        List<JobEntity> expiredJobs = jobRepository.findByStatusAndLockExpiresAtBeforeOrderByLockExpiresAtAsc(
                JobStatus.PROCESSING,
                now,
                PageRequest.of(0, properties.getRecoveryBatchSize())
        );

        int recovered = 0;
        for (JobEntity job : expiredJobs) {
            try {
                transitionFailure(job, "Processing lease expired before the worker completed the job", true, now);
                jobRepository.saveAndFlush(job);
                safeQueueSync(job, "recover");
                recovered++;
            } catch (OptimisticLockException ex) {
                log.debug("Skipped recovery for job {} because it was updated concurrently", job.getId());
            }
        }
        return recovered;
    }

    @Transactional
    public int hydrateQueuedJobs() {
        Instant cutoff = Instant.now().plus(properties.getHydrationLookAhead());
        List<JobEntity> queuedJobs = jobRepository.findByStatusAndAvailableAtBeforeOrderByAvailableAtAsc(
                JobStatus.QUEUED,
                cutoff,
                PageRequest.of(0, properties.getHydrationBatchSize())
        );

        queuedJobs.forEach(job -> safeQueueSync(job, "hydrate"));
        return queuedJobs.size();
    }

    private void transitionFailure(JobEntity job, String errorMessage, boolean retryable, Instant now) {
        int updatedRetryCount = job.getRetryCount() + 1;
        job.setRetryCount(updatedRetryCount);
        job.setLastError(errorMessage);
        job.setLockExpiresAt(null);
        job.setProcessorId(null);

        if (retryable && updatedRetryCount <= job.getMaxRetries()) {
            job.setStatus(JobStatus.QUEUED);
            job.setAvailableAt(now.plus(backoffPolicy.nextDelay(updatedRetryCount)));
            job.setCompletedAt(null);
            return;
        }

        job.setStatus(JobStatus.FAILED);
        job.setCompletedAt(now);
    }

    private boolean tryClaimWithRepair(JobEntity job, Instant now, Instant leaseUntil) {
        try {
            if (queueStore.claim(job.getId(), now, leaseUntil)) {
                return true;
            }
            queueStore.enqueue(job);
            return queueStore.claim(job.getId(), now, leaseUntil);
        } catch (DataAccessException ex) {
            throw ex;
        }
    }

    private JobEntity loadJob(String jobId) {
        return jobRepository.findById(jobId).orElseThrow(() -> new JobNotFoundException(jobId));
    }

    private void assertQueuedAndAvailable(JobEntity job, Instant now) {
        if (job.getStatus() != JobStatus.QUEUED) {
            throw new JobConflictException("Job %s is not queued".formatted(job.getId()));
        }
        if (job.getAvailableAt().isAfter(now)) {
            throw new JobConflictException("Job %s is delayed until %s".formatted(job.getId(), job.getAvailableAt()));
        }
    }

    private void assertOwnedProcessingJob(JobEntity job, String processorId) {
        if (job.getStatus() != JobStatus.PROCESSING) {
            throw new JobConflictException("Job %s is not in processing".formatted(job.getId()));
        }
        String normalizedProcessorId = processorId.trim();
        if (!normalizedProcessorId.equals(job.getProcessorId())) {
            throw new JobConflictException("Job %s is owned by a different processor".formatted(job.getId()));
        }
    }

    private void safeQueueSync(JobEntity job, String operation) {
        try {
            if (job.getStatus() == JobStatus.QUEUED) {
                queueStore.enqueue(job);
            } else if (job.getStatus() == JobStatus.PROCESSING) {
                queueStore.cacheState(job);
            } else {
                queueStore.markFailed(job);
            }
        } catch (DataAccessException ex) {
            log.warn("Redis sync failed during {} for job {}. The database state remains durable.", operation, job.getId(), ex);
        }
    }

    private void safeCacheSync(JobEntity job, String operation) {
        try {
            queueStore.cacheState(job);
        } catch (DataAccessException ex) {
            log.warn("Redis cache sync failed during {} for job {}", operation, job.getId(), ex);
        }
    }

    private JobResponse toResponse(JobEntity job) {
        return new JobResponse(
                job.getId(),
                job.getPayload(),
                job.getStatus(),
                job.getRetryCount(),
                job.getMaxRetries(),
                job.getProcessorId(),
                job.getLastError(),
                job.getAvailableAt(),
                job.getLockExpiresAt(),
                job.getCreatedAt(),
                job.getUpdatedAt(),
                job.getCompletedAt()
        );
    }
}
