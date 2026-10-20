'use strict';

// Run after insert_bank_and_series.js: node insert_question.js [path/to/config-test.toml]
const { Long } = require('mongodb');
const { SeedError, withDatabase, buildCatalog, upsertDocuments, reportFailure } = require('./insert_bank_and_series');

function buildQuestions() {
  // [题库简称, 标题, 内容, 难度, 标签]。跨题库关联仅使用相同职位方向。
  const definitions = [
    ['redis', 'Redis 的持久化机制有哪些？', 'Redis 提供 RDB 快照与 AOF 日志两种持久化方式。RDB 适合备份和快速恢复，AOF 通过追加写命令降低数据丢失窗口。需要结合恢复时间、写入压力、磁盘空间和数据可靠性选择策略。' + '实际测试应覆盖正常重启、异常断电、日志重写与备份恢复，并观察不同同步策略对延迟及吞吐量的影响。'.repeat(4), 2, ['Redis', '数据库', '持久化']],
    ['mysql', 'MySQL 索引为什么采用 B+ 树？', 'B+ 树的内部节点只保存键和指针，叶子保存数据并形成有序链，适合范围查询与磁盘页访问。', 2, ['MySQL', '数据库', '索引']],
    ['mysql', '什么是事务的 ACID 特性？', 'ACID 表示原子性、一致性、隔离性和持久性。', 1, ['MySQL', '数据库', '事务']],
    ['mysql', '什么情况下联合索引会失效？', '需要结合最左前缀、函数操作、隐式类型转换和执行计划分析。', 2, ['MySQL', '数据库', '索引']],
    ['redis', 'Redis 缓存穿透如何处理？', '可使用空值缓存、布隆过滤器和输入校验，注意过期时间和更新一致性。', 2, ['Redis', '数据库', '缓存']],
    ['redis', '缓存和数据库如何保持一致？', '可使用先更新数据库再删除缓存，并通过重试与监控处理删除失败。', 3, ['Redis', 'MySQL', '数据库', '缓存']],
    ['go', 'Go 的 goroutine 与线程有什么区别？', 'goroutine 由 Go 调度器管理，拥有可增长的栈，多路复用到操作系统线程。', 1, ['Go', '并发']],
    ['go', '如何使用 context 取消任务？', '传递 context 并监听 Done，在超时或取消时及时退出并释放资源。', 2, ['Go', '并发', 'context']],
    ['go', 'channel 有缓冲和无缓冲的区别？', '无缓冲 channel 需要收发双方同步，有缓冲 channel 在容量范围内可解耦收发。', 1, ['Go', '并发']],
    ['go', 'Go 中的 interface 是什么？', 'interface 通过方法集合描述行为，类型实现全部方法即可满足接口。', 1, ['Go', '接口']],
    ['js', 'JavaScript 事件循环如何执行？', '事件循环协调同步任务、微任务和后续任务，Promise 回调通常在微任务阶段执行。', 2, ['JavaScript', '异步']],
    ['js', 'let 与 const 有什么区别？', '两者都具有块级作用域，const 绑定不可重新赋值，但对象属性仍可能修改。', 1, ['JavaScript', '基础']],
    ['js', '如何理解闭包？', '函数保留对其词法作用域变量的访问能力，常用于封装状态。', 2, ['JavaScript', '作用域']],
    ['vue', 'Vue 响应式系统的原理是什么？', 'Vue 通过代理读写收集依赖，并在数据变化时触发对应副作用。', 2, ['Vue', 'JavaScript', '响应式']],
    ['vue', 'Vue 组件如何通信？', '可使用 props、事件、依赖注入或集中状态管理。', 1, ['Vue', '组件']],
    ['ml', '什么是过拟合？', '模型在训练集表现良好而泛化能力不足，可通过正则化与数据增强缓解。', 1, ['机器学习', '泛化']],
    ['ml', '精确率和召回率有什么区别？', '精确率衡量预测正例中的正确比例，召回率衡量实际正例被识别的比例。', 2, ['机器学习', '评估']],
    ['ml', '梯度下降如何更新参数？', '沿损失函数梯度的反方向更新参数，学习率控制每次更新幅度。', 1, ['机器学习', '优化']],
    ['mysql', '如何分析 SQL 慢查询？', '结合慢查询日志和 EXPLAIN 检查访问路径、扫描行数与索引选择。', 2, ['MySQL', '数据库', 'SQL']],
    ['redis', 'Redis 的数据类型有哪些？', '常用类型包括 String、Hash、List、Set 和 Sorted Set。', 1, ['Redis', '数据库', '数据结构']],
    ['go', '如何避免 Go 数据竞争？', '用互斥锁、channel 或原子操作建立同步关系，并使用 race 检测器辅助检查。', 3, ['Go', '并发']],
    ['js', 'Promise.all 的行为是什么？', '全部任务成功时返回结果数组，任意任务拒绝时返回的 Promise 拒绝。', 2, ['JavaScript', '异步']],
    ['vue', 'computed 和 watch 如何选择？', '派生状态适合 computed，执行异步操作或其他副作用适合 watch。', 2, ['Vue', '响应式']],
    ['ml', '训练集和测试集如何划分？', '避免信息泄漏，并按任务需要采用随机划分、分层采样或时间顺序划分。', 2, ['机器学习', '评估']],
    ['go', '如何设计 Go 缓存访问层？', '通过小接口隔离存储实现，并传递 context 控制超时和取消。', 3, ['Go', 'Redis', '数据库']],
    ['mysql', 'SQL LIKE 中特殊字符如何转义？', '关键词 a.b、Go.* 和带括号的内容应用于验证接口的字面匹配行为。', 2, ['MySQL', '数据库', 'SQL']],
    ['js', 'Unicode 字符串长度如何计算？', 'UTF-16 长度和 Unicode 字符数量可能不同，例如中文与 emoji 🙂。', 2, ['JavaScript', 'Unicode']],
    ['redis', '隐藏题目：Redis 淘汰策略', 'status=2，不应出现在列表、热门和题库正常题目数量中。', 2, ['Redis', '数据库']],
    ['mysql', '删除题目：MySQL 锁机制', 'status=3，不应出现在列表、热门和题库正常题目数量中。', 3, ['MySQL', '数据库']],
    ['ml', '隐藏题目：模型部署', 'status=2，用于验证过滤规则。', 3, ['机器学习', '部署']],
  ];
  const { banks } = buildCatalog();
  return definitions.map(([bank, title, content, difficulty, tags], index) => {
    const primary = banks.find(item => item.bank_id === `seed_bank_${bank}`);
    const bank_list = [primary.bank_id];
    if (index === 5) bank_list.push('seed_bank_mysql');
    if (index === 24) bank_list.push('seed_bank_redis');
    const created = new Date(Date.UTC(2026, 0, index + 1, 8));
    return {
      question_id: `seed_q_${String(index + 1).padStart(3, '0')}`,
      title, content, bank_list, job_name: primary.job_name, difficulty, tags,
      status: index === 28 ? 3 : index >= 27 ? 2 : 1,
      vip: false,
      // 热度最高和浏览量最高故意是不同题，用于发现错误排序。
      hot_degree: index === 0 ? 999 : index >= 27 ? 2000 + index : 500 - index * 7,
      view_count: index === 1 ? 9999 : index * 37 + 2,
      thumbs_up_count: (index * 17) % 201,
      dislike_count: (index * 3) % 19,
      order: Long.fromNumber((index + 1) * 10),
      create_time: created,
      update_time: new Date(created.getTime() + (index % 4) * 3600000),
    };
  });
}

async function checkDependencies(db, questions) {
  const bankIDs = [...new Set(questions.flatMap(question => question.bank_list))];
  const banks = await db.collection('question_bank').find(
    { bank_id: { $in: bankIDs } },
    { projection: { _id: 0, bank_id: 1, series_id: 1, job_name: 1 } },
  ).toArray();
  const byID = new Map(banks.map(bank => [bank.bank_id, bank]));
  const missing = bankIDs.filter(id => !byID.has(id));
  if (missing.length) throw new SeedError(`缺少题库 ${missing.join(', ')}。请先执行 insert_bank_and_series.js。`);
  const seriesIDs = [...new Set(banks.map(bank => bank.series_id))];
  const series = await db.collection('question_bank_series').find(
    { series_id: { $in: seriesIDs } }, { projection: { _id: 0, series_id: 1, job_name: 1 } },
  ).toArray();
  const bySeries = new Map(series.map(item => [item.series_id, item]));
  for (const question of questions) {
    for (const bankID of question.bank_list) {
      const bank = byID.get(bankID);
      const parent = bySeries.get(bank.series_id);
      if (!parent || bank.job_name !== question.job_name || parent.job_name !== question.job_name) {
        throw new SeedError(`题库 ${bankID} 的系列或职位关联不完整。请先执行 insert_bank_and_series.js。`);
      }
    }
  }
}

async function seedQuestions(db, log = console.log) {
  const questions = buildQuestions();
  await checkDependencies(db, questions);
  await upsertDocuments(db, 'question', questions, 'question_id', log);
  const normal = questions.filter(question => question.status === 1).length;
  log(`[seed] 完成：${questions.length} 道题，正常 ${normal}、隐藏 2、删除 1。`);
}

if (require.main === module) {
  withDatabase(process.argv[2], seedQuestions).catch(reportFailure);
}

module.exports = { buildQuestions, checkDependencies, seedQuestions };
