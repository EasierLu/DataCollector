<template>
  <div class="not-found-page">
    <div class="not-found-content">
      <h1>404</h1>
      <p>页面不存在</p>
      <el-button type="primary" @click="$router.push('/dashboard')">返回首页</el-button>
    </div>
  </div>
</template>

<style scoped>
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}

.not-found-content {
  text-align: center;
}

.not-found-content h1 {
  font-size: 96px;
  font-weight: 800;
  color: #4f46e5;
  margin: 0;
}

.not-found-content p {
  font-size: 18px;
  color: #6b7280;
  margin: 16px 0 32px;
}
</style>
