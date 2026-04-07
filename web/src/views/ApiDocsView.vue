<script setup lang="ts">
import { marked } from 'marked'
import apiDocsMd from '@/docs/api.md?raw'

const renderedHtml = marked(apiDocsMd) as string
</script>

<template>
  <div>
    <h2 class="page-title">API 文档</h2>
    <el-card shadow="hover">
      <div class="markdown-body" v-html="renderedHtml" />
    </el-card>
  </div>
</template>

<style scoped>
.page-title {
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 20px;
}

.markdown-body {
  line-height: 1.7;
  color: #374151;
}

:deep(.markdown-body h1) {
  font-size: 26px;
  font-weight: 700;
  color: #1f2937;
  padding-bottom: 10px;
  border-bottom: 2px solid #e5e7eb;
  margin: 0 0 20px 0;
}

:deep(.markdown-body h2) {
  font-size: 20px;
  font-weight: 700;
  color: #1f2937;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
  margin: 32px 0 16px 0;
}

:deep(.markdown-body h3) {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin: 24px 0 12px 0;
}

:deep(.markdown-body h4) {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin: 20px 0 8px 0;
}

:deep(.markdown-body p) {
  margin: 8px 0;
}

:deep(.markdown-body hr) {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 24px 0;
}

:deep(.markdown-body blockquote) {
  border-left: 4px solid #818cf8;
  padding: 8px 16px;
  margin: 12px 0;
  background: #f5f3ff;
  color: #4c1d95;
  border-radius: 0 6px 6px 0;
}

:deep(.markdown-body blockquote p) {
  margin: 4px 0;
}

:deep(.markdown-body table) {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
  font-size: 14px;
}

:deep(.markdown-body th) {
  background: #f3f4f6;
  font-weight: 600;
  text-align: left;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
}

:deep(.markdown-body td) {
  padding: 8px 12px;
  border: 1px solid #e5e7eb;
}

:deep(.markdown-body tr:nth-child(even)) {
  background: #f9fafb;
}

:deep(.markdown-body pre) {
  background: #111827;
  color: #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  margin: 12px 0;
  font-size: 13px;
  line-height: 1.6;
}

:deep(.markdown-body pre code) {
  background: transparent;
  color: inherit;
  padding: 0;
  border-radius: 0;
  font-size: inherit;
}

:deep(.markdown-body code) {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  color: #dc2626;
  font-size: 0.9em;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

:deep(.markdown-body ul),
:deep(.markdown-body ol) {
  padding-left: 24px;
  margin: 8px 0;
}

:deep(.markdown-body li) {
  margin: 4px 0;
}

:deep(.markdown-body a) {
  color: #4f46e5;
  text-decoration: none;
}

:deep(.markdown-body a:hover) {
  text-decoration: underline;
}

:deep(.markdown-body strong) {
  font-weight: 600;
  color: #1f2937;
}
</style>
