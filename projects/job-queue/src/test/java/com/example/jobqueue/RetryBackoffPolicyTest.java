package com.example.jobqueue;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Duration;

import com.example.jobqueue.config.JobQueueProperties;
import com.example.jobqueue.service.RetryBackoffPolicy;

import org.junit.jupiter.api.Test;

class RetryBackoffPolicyTest {

    @Test
    void shouldDoubleDelayUntilTheConfiguredCap() {
        JobQueueProperties properties = new JobQueueProperties();
        properties.setBaseRetryDelay(Duration.ofSeconds(10));
        properties.setMaxRetryDelay(Duration.ofMinutes(2));

        RetryBackoffPolicy policy = new RetryBackoffPolicy(properties);

        assertThat(policy.nextDelay(1)).isEqualTo(Duration.ofSeconds(10));
        assertThat(policy.nextDelay(2)).isEqualTo(Duration.ofSeconds(20));
        assertThat(policy.nextDelay(4)).isEqualTo(Duration.ofSeconds(80));
        assertThat(policy.nextDelay(8)).isEqualTo(Duration.ofMinutes(2));
    }
}
