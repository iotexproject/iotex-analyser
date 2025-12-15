-- Cleanup script to drop block_action_partition and all child partitions
-- WARNING: This will permanently delete all data in the table and its partitions

-- Method 1: Simple DROP (automatically drops all partitions)
-- This is the recommended approach - dropping the parent table automatically drops all child partitions
DROP TABLE IF EXISTS block_action_partition CASCADE;

-- Verify the table is gone
SELECT EXISTS (
    SELECT 1 FROM information_schema.tables 
    WHERE table_name = 'block_action_partition'
) as table_exists;

-- Optional: Check if any orphaned partition tables remain (should be empty after DROP CASCADE)
SELECT tablename FROM pg_tables 
WHERE tablename LIKE 'block_action_partition%'
ORDER BY tablename;
