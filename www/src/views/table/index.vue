<template>
  <div class="app-container">
    <el-table
      v-loading="listLoading"
      :data="list"
      element-loading-text="Loading"
      border
      fit
      highlight-current-row>
      <el-table-column align="center" label="ID" width="95">
        <template slot-scope="scope">
          {{ scope.ID }}
        </template>
      </el-table-column>
      <el-table-column label="Key" >
        <template slot-scope="scope">
          {{ scope.row.Key }}
        </template>
      </el-table-column>
      <el-table-column label="PusherId"  align="center">
        <template slot-scope="scope">
          <span>{{ scope.row.PusherId }}</span>
        </template>
      </el-table-column>
      <el-table-column label="RoomName"  align="center">
        <template slot-scope="scope">
          {{ scope.row.RoomName }}
        </template>
      </el-table-column>
      <el-table-column label="Source" >
        <template slot-scope="scope">
          {{ scope.row.Source }}
        </template>
      </el-table-column>
      <el-table-column align="center" prop="CreatedAt" label="CreatedAt" >
        <template slot-scope="scope">
          <i class="el-icon-time"/>
          <span>{{ scope.row.CreatedAt }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script>
  import {getList} from '@/api/table'

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
