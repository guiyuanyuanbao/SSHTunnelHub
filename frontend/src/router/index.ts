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
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '访问验证', public: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 全局路由守卫
router.beforeEach((to, _from, next) => {
  if (to.meta.title) {
    document.title = `${to.meta.title} - SSHTunnelHub`
  }

  const token = localStorage.getItem('ssh_hub_token')

  // 若访问登录页
  if (to.path === '/login') {
    if (token) {
      // 已经持有有效 token，直接重定向到仪表盘
      return next('/')
    }
    return next()
  }

  // 公开页面直接放行
  if (to.meta.public) {
    return next()
  }

  // 受保护页面: 若未登录，重定向到登录页并记录目标路由
  if (!token) {
    return next({
      path: '/login',
      query: { redirect: to.fullPath },
    })
  }

  next()
})

export default router
