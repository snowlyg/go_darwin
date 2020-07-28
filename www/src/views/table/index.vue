<template>
  <div class="app-container">
    <el-table
      v-loading="listLoading"
      :data="list"
      element-loading-text="Loading"
      border
      fit
      highlight-current-row
    >
      <el-table-column align="center" label="ID" width="95">
        <template slot-scope="scope">
          {{ scope.IDSelector0 }}
        </template>
      </el-table-column>
      <el-table-column label="Key">
        <template slot-scope="scope">
          {{ scope.row.Key }}
        </template>
      </el-table-column>
      <el-table-column label="PusherId" width="110" align="center">
        <template slot-scope="scope">
          <span>{{ scope.row.PusherId }}</span>
        </template>
      </el-table-column>
      <el-table-column label="RoomName" width="110" align="center">
        <template slot-scope="scope">
          {{ scope.row.RoomName }}
        </template>
      </el-table-column>
      <el-table-column class-name="status-col" label="Source" width="110" align="center">
        <template slot-scope="scope">
          <el-tag :type="scope.row.Source | statusFilter">{{ scope.row.Source }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column align="center" prop="CreatedAt" label="CreatedAt" width="200">
        <template slot-scope="scope">
          <i class="el-icon-time" />
          <span>{{ scope.row.CreatedAt }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script>
import { getList } from '@/api/table'

export default {
  filters: {
    statusFilter(status) {
      const statusMap = {
        published: 'success',
        draft: 'gray',
        deleted: 'danger'
      }
      return statusMap[status]
    }
  },
  data() {
    return {
      list: null,
      listLoading: true
    }
  },
  created() {
    this.fetchData()
  },
  methods: {
    fetchData() {
      this.listLoading = true
      getList().then(response => {
        this.list = response.data.items
        this.listLoading = false
      })
    }
  }
}
</script>
