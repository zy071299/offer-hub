'use strict';

// Run first: node insert_bank_and_series.js [path/to/config-test.toml]
const fs = require('node:fs');
const path = require('node:path');
const TOML = require('@iarna/toml');
const { MongoClient, Long } = require('mongodb');

class SeedError extends Error {}

function readConfig(configPath = path.resolve(__dirname, '..', 'config-test.toml')) {
  let parsed;
  try {
    parsed = TOML.parse(fs.readFileSync(configPath, 'utf8').replace(/^\uFEFF/, ''));
  } catch {
    throw new SeedError(`无法读取或解析 TOML 配置：${path.resolve(configPath)}`);
  }
  const cfg = parsed.MongoDB;
  if (!cfg || typeof cfg.url !== 'string' || !/^mongodb(?:\+srv)?:\/\//.test(cfg.url)) {
    throw new SeedError('请在配置文件 [MongoDB] 中填写有效的 url。');
  }
  if (typeof cfg.database !== 'string' || !cfg.database.trim()) {
    throw new SeedError('请在配置文件 [MongoDB] 中填写 database。');
  }
  const maxPoolSize = cfg.max_pool ?? 0;
  const minPoolSize = cfg.min_pool ?? 0;
  if (!Number.isSafeInteger(maxPoolSize) || maxPoolSize < 0 ||
      !Number.isSafeInteger(minPoolSize) || minPoolSize < 0 ||
      (maxPoolSize > 0 && minPoolSize > maxPoolSize)) {
    throw new SeedError('MongoDB max_pool / min_pool 必须为非负整数，min_pool 不得大于非零 max_pool。');
  }
  for (const key of ['user', 'pwd']) {
    if (cfg[key] !== undefined && typeof cfg[key] !== 'string') {
      throw new SeedError(`MongoDB ${key} 必须为字符串。`);
    }
  }
  const options = {
    connectTimeoutMS: 5000,
    serverSelectionTimeoutMS: 5000,
    maxPoolSize,
    minPoolSize,
  };
  // 与 Go 初始化保持一致：显式凭证优先；URI 的 authSource 等选项仍交给官方驱动解析。
  if (cfg.user || cfg.pwd) options.auth = { username: cfg.user || '', password: cfg.pwd || '' };
  return { uri: cfg.url, database: cfg.database, options };
}

async function withDatabase(configPath, action) {
  const cfg = readConfig(configPath);
  const client = new MongoClient(cfg.uri, cfg.options);
  try {
    await client.connect();
    const db = client.db(cfg.database);
    await db.command({ ping: 1 });
    console.log(`[seed] 已连接数据库 ${cfg.database}`);
    return await action(db);
  } finally {
    await client.close();
  }
}

function buildCatalog() {
  const created = new Date('2026-01-01T00:00:00Z');
  const series = [
    ['seed_series_database', '数据库', '后端开发', 10],
    ['seed_series_go', 'Go 语言', '后端开发', 20],
    ['seed_series_frontend', '前端基础', '前端开发', 30],
    ['seed_series_ai', '人工智能', 'AI开发', 40],
  ].map(([series_id, series_name, job_name, order]) => ({
    series_id, series_name, job_name, order: Long.fromNumber(order),
    create_time: new Date(created), update_time: new Date(created),
  }));
  const banks = [
    ['seed_bank_mysql', 'seed_series_database', 'MySQL题库', 'MySQL 索引、事务和 SQL', '后端开发', 10],
    ['seed_bank_redis', 'seed_series_database', 'Redis题库', 'Redis 缓存、持久化和数据结构', '后端开发', 20],
    ['seed_bank_go', 'seed_series_go', 'Go题库', 'Go 并发、内存和接口', '后端开发', 10],
    ['seed_bank_js', 'seed_series_frontend', 'JavaScript题库', 'JavaScript 基础与异步机制', '前端开发', 10],
    ['seed_bank_vue', 'seed_series_frontend', 'Vue题库', 'Vue 响应式与组件通信', '前端开发', 20],
    ['seed_bank_ml', 'seed_series_ai', '机器学习题库', '机器学习训练和评估', 'AI开发', 10],
    ['seed_bank_empty', 'seed_series_ai', '空题库', '验证 count=0 和空列表', 'AI开发', 20],
  ].map(([bank_id, series_id, bank_name, desc, job_name, order]) => ({
    bank_id, series_id, bank_name, bank_logo: '', desc, job_name,
    order: Long.fromNumber(order), create_time: new Date(created), update_time: new Date(created),
  }));
  return { series, banks };
}

function makeUpserts(documents, idField) {
  return documents.map(document => {
    const { create_time, ...fields } = document;
    return { updateOne: {
      filter: { [idField]: document[idField] },
      update: { $set: fields, $setOnInsert: { create_time } },
      upsert: true,
    } };
  });
}

async function upsertDocuments(db, collection, documents, idField, log = console.log) {
  const result = await db.collection(collection).bulkWrite(makeUpserts(documents, idField), { ordered: true });
  log(`[seed] ${collection}: 新增 ${result.upsertedCount}，匹配 ${result.matchedCount}，更新 ${result.modifiedCount}`);
}

async function seedBanksAndSeries(db, log = console.log) {
  const { series, banks } = buildCatalog();
  await upsertDocuments(db, 'question_bank_series', series, 'series_id', log);
  await upsertDocuments(db, 'question_bank', banks, 'bank_id', log);
  log(`[seed] 完成：${series.length} 个系列、${banks.length} 个题库。下一步执行 insert_question.js。`);
}

function reportFailure(error) {
  // 驱动错误可能包含连接详情，不输出 URI、凭证或原始错误堆栈。
  console.error(`[seed] 失败：${error instanceof SeedError ? error.message : 'MongoDB 操作失败，请检查连接、认证、写权限及已有文档的字段类型。'}`);
  process.exitCode = 1;
}

if (require.main === module) {
  withDatabase(process.argv[2], seedBanksAndSeries).catch(reportFailure);
}

module.exports = { SeedError, readConfig, withDatabase, buildCatalog, makeUpserts, upsertDocuments, seedBanksAndSeries, reportFailure };
