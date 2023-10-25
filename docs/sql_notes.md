- 获取指定高度下所有 native staking bucket 的基本信息
```sql
```sql
SELECT
  id,
  block_height,
  bucket_id,
  owner_address,
  candidate,
  (SELECT SUM(b.amount) FROM staking_actions b WHERE b.block_height <= 5161480 AND b.bucket_id = a.bucket_id) AS amount,
  act_type,
  auto_stake,
  duration
FROM
  staking_actions a
WHERE
  id = ANY (ARRAY(SELECT MAX(id) FROM staking_actions WHERE block_height <= 5161480 GROUP BY bucket_id)))
```
- 翻页查询优化
```sql
SELECT *
FROM block_action
WHERE contract_address='io154mvzs09vkgn0hw6gg3ayzw5w39jzp47f8py9v'
ORDER BY  id ASC limit 10 offset 1600000; 
```
上面offset 太大，可以使用下面的 sql 来优化
```sql
SELECT *
FROM block_action
WHERE contract_address='io154mvzs09vkgn0hw6gg3ayzw5w39jzp47f8py9v'
AND id>=1675572 ORDER BY  id ASC limit 10;
```
- PG_hint_plan 中带有 schema 的表名，需要使用别名
```sql
/*+ IndexScan(a) */select * from reader.block_receipts a where action_hash='3866a5be503847400594d1911e5a411a83fc8b10f1ecd1aef4cb89bee7beeb94';
```
- 修复自增 Sequence之存储过程版
```sql
CREATE OR REPLACE FUNCTION fix_sequences() RETURNS void AS $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN (SELECT table_name, column_name FROM information_schema.columns WHERE column_default LIKE 'nextval%') LOOP
    EXECUTE format('SELECT setval(pg_get_serial_sequence(''%s'', ''%s''), (SELECT MAX(%s) FROM %s));', r.table_name, r.column_name, r.column_name, r.table_name);
  END LOOP;
END;
$$ LANGUAGE plpgsql;
#执行
SELECT fix_sequences();
```

- 调整 Schema 的搜索路径, 优先使用 reader schema,若 reader schema 中没有则使用 public schema
```sql
SELECT pg_catalog.set_config('search_path', 'reader,public', false);
```
- 获取staking_buckets 表中指定高度下所有 bucket_id 的最新记录
```sql
WITH max_ids AS (
    SELECT MAX(id) AS max_id
    FROM "staking_buckets"
    WHERE block_height <= 20924943
    GROUP BY bucket_id
)
SELECT *
FROM staking_buckets t1
INNER JOIN max_ids t2 ON t2.max_id = t1.id
```