-- Supports exporting up to a snapshot cutoff height H to avoid conflicts

-- ============================================================================
-- PHASE 1: Pre-migration setup and optimization
-- ============================================================================

-- Disable foreign key checks if any (speeds up bulk insert)
-- Note: block_action_partition should not have FK constraints, but good practice

-- Create a temporary table to hold batches (optional, for staging)
-- This allows us to verify data before final insert
CREATE TEMPORARY TABLE IF NOT EXISTS batch_staging AS
SELECT * FROM block_action LIMIT 0;

-- Determine snapshot cutoff height H (the highest block_height to export)
-- If a custom value is provided via GUC `iotex.migrate.cutoff_height`, it will be used.
-- Otherwise, it will capture current MAX(block_height) as H.
DO $$
DECLARE
    v_h BIGINT;
    v_cfg TEXT;
BEGIN
    v_cfg := current_setting('iotex.migrate.cutoff_height', true);
    IF v_cfg IS NULL OR v_cfg = '' THEN
        SELECT COALESCE(MAX(block_height), 0) INTO v_h FROM block_action;
        PERFORM set_config('iotex.migrate.cutoff_height', v_h::text, false);
        RAISE NOTICE 'Snapshot cutoff height H auto-detected: %', v_h;
    ELSE
        v_h := v_cfg::BIGINT;
        PERFORM set_config('iotex.migrate.cutoff_height', v_h::text, false);
        RAISE NOTICE 'Snapshot cutoff height H provided: %', v_h;
    END IF;
END $$;

-- ============================================================================
-- PHASE 2: Bulk migration with batching
-- ============================================================================

-- Option A: Direct batch migration (recommended for 250M rows)
-- Migrate in chunks to avoid memory issues and long locks
-- Adjust batch size based on available memory (10M rows per batch is safe)

DO $$
DECLARE
    -- Batch size controls per-iteration rows; use smaller default for stability
    -- You can override via: SET iotex.migrate.batch_size = '1000000';
    v_batch_size INTEGER := COALESCE(NULLIF(current_setting('iotex.migrate.batch_size', true), '')::INTEGER, 10000);  -- default 10k rows per batch
    v_total_rows BIGINT;
    v_processed BIGINT := 0;
    v_start_time TIMESTAMP;
    v_batch_start_id BIGINT := 0;
    v_batch_end_id BIGINT;
    v_h BIGINT := current_setting('iotex.migrate.cutoff_height')::BIGINT;
BEGIN
    v_start_time := CLOCK_TIMESTAMP();
    
    -- Get total row count
    SELECT COUNT(*) INTO v_total_rows FROM block_action WHERE block_height <= v_h;
    RAISE NOTICE 'Total rows to migrate: %', v_total_rows;
    
    -- Get max ID
    SELECT MAX(id) INTO v_batch_end_id FROM block_action WHERE block_height <= v_h;
    
    -- Migrate in batches
    WHILE v_batch_start_id <= v_batch_end_id LOOP
        RAISE NOTICE 'Migrating batch: % to % (% of % rows processed)', 
            v_batch_start_id, 
            LEAST(v_batch_start_id + v_batch_size, v_batch_end_id),
            v_processed,
            v_total_rows;
        
        INSERT INTO block_action_partition (
            id, action_hash, action_type, block_height,
            sender, recipient, gas_price, gas_limit, nonce, amount,
            gas_consumed, chain_id, encoding, version,
            contract_address, status, execution_revert_msg, payload, timestamp
        )
        SELECT
            id, action_hash, action_type, block_height,
            sender, recipient, gas_price, gas_limit, nonce, amount,
            gas_consumed, chain_id, encoding, version,
            contract_address, status, execution_revert_msg, payload, timestamp
                FROM block_action
                WHERE block_height <= v_h
                    AND id > v_batch_start_id AND id <= v_batch_start_id + v_batch_size
        ORDER BY block_height;
        
        v_processed := v_processed + v_batch_size;
        v_batch_start_id := v_batch_start_id + v_batch_size;
        
        -- Optional: Force checkpoint every batch to free up WAL logs
        CHECKPOINT;
        
        -- Show progress
        RAISE NOTICE 'Batch completed in %. Elapsed time: %', 
            (SELECT EXTRACT(EPOCH FROM (CLOCK_TIMESTAMP() - v_start_time))::INTEGER || ' seconds'),
            CLOCK_TIMESTAMP() - v_start_time;
    END LOOP;
    
    RAISE NOTICE 'Migration completed in: %', CLOCK_TIMESTAMP() - v_start_time;
END $$;

-- ============================================================================
-- PHASE 3: Verification
-- ============================================================================

-- Verify row counts match
DO $$
DECLARE
    v_source_count BIGINT;
    v_target_count BIGINT;
BEGIN
    SELECT COUNT(*) INTO v_source_count FROM block_action WHERE block_height <= current_setting('iotex.migrate.cutoff_height')::BIGINT;
    SELECT COUNT(*) INTO v_target_count FROM block_action_partition;
    
    RAISE NOTICE 'Source (block_action): % rows', v_source_count;
    RAISE NOTICE 'Target (block_action_partition): % rows', v_target_count;
    
    IF v_source_count = v_target_count THEN
        RAISE NOTICE '✓ Verification PASSED: Row counts match!';
    ELSE
        RAISE WARNING '✗ Verification FAILED: Row count mismatch! Difference: %', 
            (v_source_count - v_target_count);
    END IF;
END $$;

-- ============================================================================
-- PHASE 4: Partition distribution analysis
-- ============================================================================

-- Check partition distribution
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as partition_size,
    (SELECT COUNT(*) FROM pg_class WHERE relname = tablename) as exists
FROM pg_tables
WHERE tablename LIKE 'block_action_partition_%'
ORDER BY tablename;

-- Summary of partitions
SELECT
    'Total partitions' as metric,
    COUNT(*) as value
FROM pg_tables
WHERE tablename LIKE 'block_action_partition_%';

-- ============================================================================
-- PHASE 5: Performance tuning (Optional, for frequently accessed partitions)
-- ============================================================================

-- Analyze the parent table for query planner optimization
ANALYZE block_action_partition;

-- Vacuum to reclaim space
VACUUM ANALYZE block_action_partition;

-- ============================================================================
-- Migration complete! Run verification below:
-- ============================================================================

-- Final summary
DO $$
DECLARE
    v_source_count BIGINT;
    v_target_count BIGINT;
    v_total_size TEXT;
BEGIN
    SELECT COUNT(*) INTO v_source_count FROM block_action WHERE block_height <= current_setting('iotex.migrate.cutoff_height')::BIGINT;
    SELECT COUNT(*) INTO v_target_count FROM block_action_partition;
    
    SELECT pg_size_pretty(pg_total_relation_size('block_action_partition')) 
    INTO v_total_size;
    
    RAISE NOTICE '';
    RAISE NOTICE '╔════════════════════════════════════════════╗';
    RAISE NOTICE '║  Migration Summary                         ║';
    RAISE NOTICE '╠════════════════════════════════════════════╣';
    RAISE NOTICE '║ Source rows:        %', RPAD(v_source_count::TEXT, 30) || '║';
    RAISE NOTICE '║ Target rows:        %', RPAD(v_target_count::TEXT, 30) || '║';
    RAISE NOTICE '║ Total size:         %', RPAD(v_total_size, 30) || '║';
    RAISE NOTICE '║ Status:             %', CASE 
        WHEN v_source_count = v_target_count THEN RPAD('✓ SUCCESS', 30)
        ELSE RPAD('✗ MISMATCH', 30)
    END || '║';
    RAISE NOTICE '╚════════════════════════════════════════════╝';
    RAISE NOTICE '';
END $$;
