package com.example.jobqueue.persistence;

import java.time.Instant;
import java.util.List;

import com.example.jobqueue.model.JobStatus;

import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;

public interface JobRepository extends JpaRepository<JobEntity, String> {

    List<JobEntity> findByStatusAndLockExpiresAtBeforeOrderByLockExpiresAtAsc(
            JobStatus status,
            Instant lockExpiresAt,
            Pageable pageable
    );

    List<JobEntity> findByStatusAndAvailableAtBeforeOrderByAvailableAtAsc(
            JobStatus status,
            Instant availableAt,
            Pageable pageable
    );
}
