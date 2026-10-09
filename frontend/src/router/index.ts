import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Hosts from '../views/Hosts.vue'
import Tunnels from '../views/Tunnels.vue'
import Logs from '../views/Logs.vue'

const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard,
    meta: { title: '运行概览' },
  },
  {
    path: '/hosts',
    name: 'Hosts',
    component: Hosts,
    meta: { title: '主机管理' },
  },
  {
    path: '/tunnels',
    name: 'Tunnels',
    component: Tunnels,
    meta: { title: '隧道管理' },
  },
  {
    path: '/logs',
    name: 'Logs',
    component: Logs,
    meta: { title: '运行日志' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
