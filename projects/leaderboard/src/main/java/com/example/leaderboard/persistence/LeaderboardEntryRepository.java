package com.example.leaderboard.persistence;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;

public interface LeaderboardEntryRepository extends JpaRepository<LeaderboardEntryEntity, String> {

    Page<LeaderboardEntryEntity> findAllByOrderByScoreDescPlayerIdAsc(Pageable pageable);

    long countByScoreGreaterThan(long score);

    long countByScoreAndPlayerIdLessThan(long score, String playerId);
}
