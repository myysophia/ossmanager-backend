# OSS管理系统接口压测方案

## 压测目标

对OSS管理系统的核心接口进行压力测试，评估系统在高并发场景下的性能表现，确保系统具备足够的稳定性和可扩展性。

## 压测工具

本方案主要使用 [k6](https://k6.io/) 作为压测工具，k6是一个现代化的负载测试工具，支持HTTP/HTTPS、WebSocket等协议的压测。

### 安装k6

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# CentOS/RHEL
sudo yum install https://dl.k6.io/rpm/repo.rpm
sudo yum install k6
```

## 压测指标

- **平均响应时间 (Average Response Time)**: 所有请求的平均响应时间
- **请求/秒 (RPS)**: 每秒处理的请求数
- **错误率 (Error Rate)**: 请求失败的百分比
- **95/99百分位响应时间**: 95%/99%的请求能在这个时间内得到响应
- **资源使用率**: CPU、内存、网络I/O等资源使用情况

## 压测场景

### 1. 认证接口压测

```javascript
// test/k6/auth_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

// 定义指标
const loginErrors = new Counter('login_errors');

// 定义压测配置
export const options = {
  stages: [
    { duration: '1m', target: 50 }, // 逐步增加到50个并发用户
    { duration: '3m', target: 50 }, // 保持50个并发用户3分钟
    { duration: '1m', target: 0 },  // 逐步减少到0个并发用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95%的请求响应时间<500ms
    'login_errors': ['count<10'],     // 登录错误次数<10
  },
};

// 测试逻辑
export default function() {
  const url = 'http://localhost:8080/api/v1/auth/login';
  const payload = JSON.stringify({
    username: `user_${__VU}`, // 使用虚拟用户ID作为用户名，确保不同用户登录
    password: 'password123',
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  // 发送登录请求
  const res = http.post(url, payload, params);
  
  // 检查响应
  const success = check(res, {
    'login successful': (r) => r.status === 200,
    'token received': (r) => JSON.parse(r.body).data && JSON.parse(r.body).data.token,
  });
  
  if (!success) {
    loginErrors.add(1);
    console.log(`Login failed: ${res.status} ${res.body}`);
  }
  
  sleep(1);
}
```

### 2. 文件上传接口压测

```javascript
// test/k6/file_upload_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// 定义指标
const uploadErrors = new Counter('upload_errors');
const uploadTime = new Trend('upload_time');

// 预先登录获取token
const users = new SharedArray('users', function() {
  // 这个函数在初始化阶段只执行一次
  const res = http.post('http://localhost:8080/api/v1/auth/login', 
    JSON.stringify({ username: 'admin', password: 'admin123' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  return [{ token: JSON.parse(res.body).data.token }];
});

// 定义压测配置
export const options = {
  stages: [
    { duration: '1m', target: 10 },  // 逐步增加到10个并发用户
    { duration: '3m', target: 10 },  // 保持10个并发用户3分钟
    { duration: '1m', target: 0 },   // 逐步减少到0个并发用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<3000'], // 95%的请求响应时间<3s
    'upload_errors': ['count<5'],      // 上传错误次数<5
  },
};

// 测试逻辑
export default function() {
  const token = users[0].token;
  
  // 创建二进制数据作为文件内容（约100KB）
  const fileContent = new Array(100 * 1024).fill('A').join('');
  
  // 创建FormData格式的请求
  const data = {
    file: http.file(fileContent, `test_${__VU}_${Date.now()}.txt`, 'text/plain'),
    config_id: '1',
  };
  
  // 发送文件上传请求
  const startTime = new Date();
  const res = http.post('http://localhost:8080/api/v1/oss/files', data, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });
  const endTime = new Date();
  
  // 记录上传时间
  uploadTime.add(endTime - startTime);
  
  // 检查响应
  const success = check(res, {
    'upload successful': (r) => r.status === 200,
    'file info received': (r) => JSON.parse(r.body).data && JSON.parse(r.body).data.download_url,
  });
  
  if (!success) {
    uploadErrors.add(1);
    console.log(`Upload failed: ${res.status} ${res.body}`);
  }
  
  sleep(3);
}
```

### 3. 文件列表查询接口压测

```javascript
// test/k6/file_list_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// 定义指标
const queryErrors = new Counter('query_errors');
const queryTime = new Trend('query_time');

// 预先登录获取token
const users = new SharedArray('users', function() {
  const res = http.post('http://localhost:8080/api/v1/auth/login', 
    JSON.stringify({ username: 'admin', password: 'admin123' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  return [{ token: JSON.parse(res.body).data.token }];
});

// 定义压测配置
export const options = {
  stages: [
    { duration: '1m', target: 30 },  // 逐步增加到30个并发用户
    { duration: '3m', target: 30 },  // 保持30个并发用户3分钟
    { duration: '1m', target: 0 },   // 逐步减少到0个并发用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95%的请求响应时间<500ms
    'query_errors': ['count<5'],      // 查询错误次数<5
  },
};

// 测试逻辑
export default function() {
  const token = users[0].token;
  
  // 随机选择页码和每页数量
  const page = Math.floor(Math.random() * 5) + 1;
  const pageSize = 10;
  
  // 发送查询请求
  const startTime = new Date();
  const res = http.get(`http://localhost:8080/api/v1/oss/files?page=${page}&page_size=${pageSize}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });
  const endTime = new Date();
  
  // 记录查询时间
  queryTime.add(endTime - startTime);
  
  // 检查响应
  const success = check(res, {
    'query successful': (r) => r.status === 200,
    'file list received': (r) => {
      const data = JSON.parse(r.body).data;
      return data && Array.isArray(data.items);
    },
  });
  
  if (!success) {
    queryErrors.add(1);
    console.log(`Query failed: ${res.status} ${res.body}`);
  }
  
  sleep(1);
}
```

### 4. 下载链接生成接口压测

```javascript
// test/k6/download_url_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// 定义指标
const urlGenErrors = new Counter('url_gen_errors');
const urlGenTime = new Trend('url_gen_time');

// 预先登录获取token和文件列表
const testData = new SharedArray('testData', function() {
  // 登录获取token
  const loginRes = http.post('http://localhost:8080/api/v1/auth/login', 
    JSON.stringify({ username: 'admin', password: 'admin123' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  const token = JSON.parse(loginRes.body).data.token;
  
  // 获取文件列表
  const listRes = http.get('http://localhost:8080/api/v1/oss/files?page=1&page_size=20', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  const files = JSON.parse(listRes.body).data.items;
  return {
    token: token,
    fileIds: files.map(file => file.id)
  };
});

// 定义压测配置
export const options = {
  stages: [
    { duration: '1m', target: 50 },  // 逐步增加到50个并发用户
    { duration: '3m', target: 50 },  // 保持50个并发用户3分钟
    { duration: '1m', target: 0 },   // 逐步减少到0个并发用户
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'], // 95%的请求响应时间<300ms
    'url_gen_errors': ['count<5'],    // 生成URL错误次数<5
  },
};

// 测试逻辑
export default function() {
  if (!testData.fileIds || testData.fileIds.length === 0) {
    console.log('No files found in database, skipping test');
    sleep(1);
    return;
  }
  
  // 随机选择一个文件ID
  const fileId = testData.fileIds[Math.floor(Math.random() * testData.fileIds.length)];
  
  // 发送生成下载链接请求
  const startTime = new Date();
  const res = http.get(`http://localhost:8080/api/v1/oss/files/${fileId}/download`, {
    headers: {
      'Authorization': `Bearer ${testData.token}`,
    },
  });
  const endTime = new Date();
  
  // 记录生成时间
  urlGenTime.add(endTime - startTime);
  
  // 检查响应
  const success = check(res, {
    'url generation successful': (r) => r.status === 200,
    'download url received': (r) => {
      const data = JSON.parse(r.body).data;
      return data && data.download_url;
    },
  });
  
  if (!success) {
    urlGenErrors.add(1);
    console.log(`URL generation failed: ${res.status} ${res.body}`);
  }
  
  sleep(1);
}
```

### 5. 混合压测场景

```javascript
// test/k6/mixed_test.js
import { group, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

// 定义指标
const errors = new Counter('errors');

// 预先登录获取token
const users = new SharedArray('users', function() {
  const res = http.post('http://localhost:8080/api/v1/auth/login', 
    JSON.stringify({ username: 'admin', password: 'admin123' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  return [{ token: JSON.parse(res.body).data.token }];
});

// 定义压测配置
export const options = {
  scenarios: {
    // 查询场景 - 高并发，占70%的流量
    queries: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 35 },
        { duration: '3m', target: 35 },
        { duration: '1m', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'queryFiles',
    },
    // 上传场景 - 中等并发，占20%的流量
    uploads: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 10 },
        { duration: '3m', target: 10 },
        { duration: '1m', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'uploadFile',
    },
    // 下载链接生成场景 - 低并发，占10%的流量
    downloads: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 5 },
        { duration: '3m', target: 5 },
        { duration: '1m', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'generateDownloadUrl',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95%的请求响应时间<1s
    'errors': ['count<50'],           // 总错误数<50
  },
};

// 文件查询场景
export function queryFiles() {
  const token = users[0].token;
  
  group('List Files', function() {
    const page = Math.floor(Math.random() * 5) + 1;
    const res = http.get(`http://localhost:8080/api/v1/oss/files?page=${page}&page_size=10`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    
    const success = check(res, {
      'list files successful': (r) => r.status === 200,
    });
    
    if (!success) errors.add(1);
  });
  
  sleep(Math.random() * 2);
}

// 文件上传场景
export function uploadFile() {
  const token = users[0].token;
  
  group('Upload File', function() {
    // 创建一个小文件（10KB左右）
    const fileContent = new Array(10 * 1024).fill('A').join('');
    const data = {
      file: http.file(fileContent, `test_${__VU}_${Date.now()}.txt`, 'text/plain'),
      config_id: '1',
    };
    
    const res = http.post('http://localhost:8080/api/v1/oss/files', data, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    
    const success = check(res, {
      'upload file successful': (r) => r.status === 200,
    });
    
    if (!success) errors.add(1);
  });
  
  sleep(Math.random() * 3 + 2);
}

// 生成下载链接场景
export function generateDownloadUrl() {
  const token = users[0].token;
  
  group('Generate Download URL', function() {
    // 首先获取文件列表
    const listRes = http.get('http://localhost:8080/api/v1/oss/files?page=1&page_size=5', {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    
    const success = check(listRes, {
      'list files successful': (r) => r.status === 200,
    });
    
    if (!success) {
      errors.add(1);
      return;
    }
    
    const files = JSON.parse(listRes.body).data.items;
    if (!files || files.length === 0) {
      console.log('No files found, skipping download URL generation');
      return;
    }
    
    // 随机选择一个文件
    const fileId = files[Math.floor(Math.random() * files.length)].id;
    
    // 生成下载链接
    const dlRes = http.get(`http://localhost:8080/api/v1/oss/files/${fileId}/download`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    
    const dlSuccess = check(dlRes, {
      'generate download url successful': (r) => r.status === 200,
    });
    
    if (!dlSuccess) errors.add(1);
  });
  
  sleep(Math.random() * 2 + 1);
}
```

## 压测执行

### 基本命令

```bash
# 执行单一场景压测
k6 run test/k6/auth_test.js

# 执行混合场景压测
k6 run test/k6/mixed_test.js

# 指定虚拟用户数和持续时间
k6 run --vus 50 --duration 30s test/k6/file_list_test.js

# 保存结果到JSON文件
k6 run --out json=result.json test/k6/file_upload_test.js
```

### 输出结果到InfluxDB和Grafana（可选）

如果需要更好的可视化，可以将结果输出到InfluxDB，然后使用Grafana展示：

```bash
# 启动InfluxDB和Grafana（使用Docker）
docker-compose up -d influxdb grafana

# 执行压测并输出到InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 test/k6/mixed_test.js
```

docker-compose.yml 配置示例：

```yaml
version: '3'
services:
  influxdb:
    image: influxdb:1.8
    ports:
      - "8086:8086"
    environment:
      - INFLUXDB_DB=k6
      - INFLUXDB_ADMIN_USER=admin
      - INFLUXDB_ADMIN_PASSWORD=admin
    volumes:
      - influxdb-data:/var/lib/influxdb

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - grafana-data:/var/lib/grafana
    depends_on:
      - influxdb

volumes:
  influxdb-data:
  grafana-data:
```

## 压测结果分析

### 预期目标

| 接口 | 并发用户数 | 平均响应时间 | 95%响应时间 | RPS | 错误率 |
|------|------------|--------------|-------------|-----|--------|
| 登录 | 50         | <200ms       | <500ms      | >100| <1%    |
| 文件上传 | 10     | <1s          | <3s         | >5  | <2%    |
| 文件列表 | 30     | <200ms       | <500ms      | >50 | <1%    |
| 下载链接 | 50     | <100ms       | <300ms      | >100| <1%    |

### 性能瓶颈判断

在压测过程中，可以通过以下指标判断系统的瓶颈：

1. **CPU 使用率**: 如果 CPU 使用率接近 100%，则可能是应用程序逻辑或并发处理能力存在瓶颈
2. **内存使用率**: 如果内存使用持续增长，可能存在内存泄漏
3. **I/O 等待**: 如果 I/O 等待时间较长，可能是数据库或文件系统存在瓶颈
4. **网络流量**: 如果网络带宽接近饱和，可能需要优化网络或考虑CDN加速

### 优化建议

根据压测结果，可能的优化方向包括：

1. **连接池优化**: 调整数据库连接池大小，确保高并发场景下有足够的连接可用
2. **缓存策略**: 对频繁访问的数据（如文件列表、文件元数据）进行缓存
3. **异步处理**: 将文件上传、MD5计算等耗时操作改为异步处理
4. **负载均衡**: 在多实例部署时，使用负载均衡分散请求压力
5. **资源限制**: 对单个用户的请求频率和并发数进行限制，防止恶意攻击

## 压测注意事项

1. **测试数据准备**: 确保数据库中有足够的测试数据，特别是文件记录
2. **隔离环境**: 在非生产环境进行压测，避免影响正常业务
3. **监控系统资源**: 在压测过程中监控系统资源使用情况
4. **逐步增加负载**: 从小负载开始，逐步增加，找到系统的极限
5. **清理测试数据**: 测试完成后清理测试产生的数据

## 参考资料

- [k6 官方文档](https://k6.io/docs/)
- [Grafana k6 Dashboard](https://grafana.com/grafana/dashboards/2587)
- [性能测试最佳实践](https://k6.io/docs/testing-guides/api-load-testing) 