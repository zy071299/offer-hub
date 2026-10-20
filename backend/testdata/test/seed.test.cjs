'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { Long } = require('mongodb');
const { buildCatalog, readConfig, makeUpserts, seedBanksAndSeries } = require('../insert_bank_and_series');
const { buildQuestions, checkDependencies, seedQuestions } = require('../insert_question');

test('fixtures match Mongo schema and cover list, metadata, details, heat and pagination', () => {
  const { series, banks } = buildCatalog();
  const questions = buildQuestions();
  assert.equal(series.length, 4);
  assert.equal(banks.length, 7);
  assert.equal(questions.length, 30);
  for (const [docs, field] of [[series, 'series_id'], [banks, 'bank_id'], [questions, 'question_id']]) {
    assert.equal(new Set(docs.map(d => d[field])).size, docs.length);
    for (const doc of docs) {
      assert.match(doc[field], /^seed_/);
      assert.ok(Long.isLong(doc.order));
      assert.ok(doc.create_time instanceof Date);
      assert.ok(doc.update_time instanceof Date);
    }
  }
  for (const bank of banks) {
    const parent = series.find(s => s.series_id === bank.series_id);
    assert.ok(parent);
    assert.equal(parent.job_name, bank.job_name);
    assert.equal(typeof bank.desc, 'string');
  }
  const expectedFields = ['question_id', 'title', 'content', 'bank_list', 'job_name', 'difficulty', 'tags', 'status', 'vip', 'hot_degree', 'view_count', 'thumbs_up_count', 'dislike_count', 'order', 'create_time', 'update_time'].sort();
  for (const question of questions) {
    assert.deepEqual(Object.keys(question).sort(), expectedFields);
    assert.ok([1, 2, 3].includes(question.status));
    assert.ok([1, 2, 3].includes(question.difficulty));
    assert.equal(typeof question.vip, 'boolean');
    for (const bankID of question.bank_list) {
      const bank = banks.find(b => b.bank_id === bankID);
      assert.ok(bank);
      assert.equal(bank.job_name, question.job_name);
    }
    for (const field of ['hot_degree','view_count','thumbs_up_count','dislike_count']) assert.ok(Number.isInteger(question[field]));
  }
  assert.ok(questions.filter(q => q.status === 1).length > 20);
  assert.ok(questions.some(q => q.status === 2));
  assert.ok(questions.some(q => q.status === 3));
  assert.ok(questions.some(q => q.bank_list.length > 1));
  assert.ok(questions.some(q => [...q.content].length > 150));
  assert.equal(questions.filter(q => q.status === 1).reduce((a,b) => a.hot_degree > b.hot_degree ? a : b).question_id, 'seed_q_001');
  assert.equal(questions.reduce((a,b) => a.view_count > b.view_count ? a : b).question_id, 'seed_q_002');
  assert.ok(!questions.some(q => q.bank_list.includes('seed_bank_empty')));
  assert.ok(questions.some(q => q.status !== 1 && q.hot_degree > 999));
});

test('upserts target stable business IDs and preserve create_time', () => {
  const questions = buildQuestions();
  const operations = makeUpserts(questions, 'question_id');
  assert.deepEqual(operations, makeUpserts(buildQuestions(), 'question_id'));
  operations.forEach((operation, i) => {
    const { filter, update, upsert } = operation.updateOne;
    assert.deepEqual(filter, { question_id: questions[i].question_id });
    assert.equal(upsert, true);
    assert.deepEqual(update.$setOnInsert, { create_time: questions[i].create_time });
    assert.equal(update.$set.create_time, undefined);
    assert.equal(update.$set._id, undefined);
    assert.equal(update.$set.question_id, questions[i].question_id);
  });
});

test('config reads actual TOML values and validates pool without connecting', () => {
  const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'offer-hub-seed-'));
  const file = path.join(folder, 'test.toml');
  try {
    fs.writeFileSync(file, '[MongoDB]\nurl="mongodb://localhost:27017/?authSource=admin"\nuser="seed-user"\npwd="p@ss"\ndatabase="seed_test"\nmax_pool=10\nmin_pool=1\n');
    const config = readConfig(file);
    assert.equal(config.database, 'seed_test');
    assert.equal(config.options.auth.username, 'seed-user');
    assert.equal(config.options.auth.password, 'p@ss');
    assert.equal(config.options.maxPoolSize, 10);
    assert.equal(config.options.minPoolSize, 1);
    assert.match(config.uri, /authSource=admin/);
    fs.writeFileSync(file, '[MongoDB]\nurl=""\ndatabase=""\n');
    assert.throws(() => readConfig(file), /MongoDB/);
    fs.writeFileSync(file, '[MongoDB]\nurl="mongodb://localhost"\ndatabase="test"\nmax_pool=2\nmin_pool=3\n');
    assert.throws(() => readConfig(file), /min_pool/);
  } finally { fs.rmSync(folder, { recursive: true, force: true }); }
});

test('missing parent records stop all question writes and parent writes precede banks', async () => {
  const order = [];
  const catalogDB = { collection: name => ({ bulkWrite: async () => { order.push(name); return { upsertedCount: 1, matchedCount: 0, modifiedCount: 0 }; } }) };
  await seedBanksAndSeries(catalogDB, () => {});
  assert.deepEqual(order, ['question_bank_series', 'question_bank']);
  let writes = 0;
  const missingDB = { collection: () => ({ find: () => ({ toArray: async () => [] }), bulkWrite: async () => { writes++; } }) };
  await assert.rejects(checkDependencies(missingDB, buildQuestions()), /insert_bank_and_series/);
  await assert.rejects(seedQuestions(missingDB, () => {}), /insert_bank_and_series/);
  assert.equal(writes, 0);
});

