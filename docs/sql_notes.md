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
- 获取 system_staking_buckets 表中指定高度下所有 bucket_id 的最新记录
```sql
WITH max_ids AS (
		SELECT MAX(id) AS max_id
		FROM system_staking_buckets
		WHERE block_height <= 1697819060
		GROUP BY bucket_id
	)
	SELECT *
	FROM system_staking_buckets t1
	RIGHT JOIN max_ids t2 ON  t1.id=t2.max_id order by bucket_id
```
- 找出 reward 不是 hermes 发放的记录,用于过滤 lsd buckets 不是 hermes 发放的记录
```sql
select name,reward_address,owner_address from delegate where reward_address<>'io12mgttmfa2ffn9uqvn0yn37f4nz43d248l2ga85' and owner_address in ('io1jzdxuv7etyfvs7th7jwyspswr6y660zjd7lykg','io1gkgraytjgx8y6p6z8wxunt6g8k52wcuf6utqwn','io1x3us2fktfq6tnftjzwtvgxh3ymvcfwy9fts7td','io1nchct6w88fa5qwsqk8sqhkn2x27psf7tn3hqqy','io1xf7ytvlysn0tmfvr2canyvkywmlu5jpch2h9qg','io105wg42thd9hajwx296473kd7wdvxj3yxs3enzx','io1yfymptak4uxzmg5ag6nyw9zwum5ffacqflntjr','io10vlzd74rhl93j9m0z95nkg3ujdgn4et9xy4qec','io1yzaq4rfp0tkdf9lft4s9xlhmlhwu65yaz8dxma','io1uxmfklwyqvjrxwpcljgqq759yy8xv26g5pt75v','io13eyqzfvluqz4zq72yr6xwlls9w38dggw7cncum','io1k4gmu8uha4qgq84zyf7vdesqahm5wnwadx5mwt','io18unn0zd4vqy8h2fmdv33tr29hpkah005w7wn86','io1ndxapwqwf7m2sn626c7q23d3lkzhptljc05e2e','io1fcj9cjhnw7ujdmw66gs294055qcn6a06xtv43f','io1aep9j4yf99uma9vuq3s32vp0qgycd8rhtsf3vz','io1s7yejy7ytuesg8se3snvnhq283y9d6qnvwseek','io19ufumht0nlz7hkrct6dmur982jymxx0em5ntzj','io1zl3yymcdxhvcrlhqhk05vqdugp732d9l3k0xur','io1er7jrwru70vsyqw5v0xzl9k8lpsxtcf2ae66l3','io1c2cacn26mawwg0vpx2ptnegg600q5kpmv75np0','io19czfrdyjt67g9tuhmmcjkuh2m9qxzv5nqyve9p','io183v7vftj3e4h76z5f5qpswhnn5737rrwjkhyds','io1qpkl9ta2l57lrespe9tpuk3ldw3n8lvdg659ye','io1h7kpk8zwqv7fgan82nmvgqwtr28l8ud659uuz6','io1xu84e32gcd9kznvs2tkp07d3fvxd2mgnhu8r28','io1mjy60gywe6spqh6sn2vjugyq6r0epddc5sgz6g','io1yaygpa4yncnj6q228znvhac8ghmch6tunyh3zv','io1d8j43c704njp2l039p96ht4q2ycjznjd9lypa8','io10utxlnxp99e0yfkunngl4jhe2saw9f3elnfngn')
```