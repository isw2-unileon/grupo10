import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '@/views/HomeView.vue'
import { useAuthStore } from '@/stores/auth'
import CalendarView from '@/views/CalendarView.vue'
// Per-route auth metadata, consumed by the navigation guard below.
declare module 'vue-router' {
  interface RouteMeta {
    // Route can only be seen by authenticated users (redirects to /login).
    requiresAuth?: boolean
    // Route is only for guests; authenticated users are sent to /home.
    guestOnly?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/login',
      name: 'login',
      meta: { guestOnly: true },
      // Lazy-loaded: split into its own chunk and only fetched when visited.
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/register',
      name: 'register',
      meta: { guestOnly: true },
      component: () => import('@/views/RegisterView.vue'),
    },
    {
      path: '/calendar',
      name: 'calendar',
      meta: { requiresAuth: true },
      component: CalendarView,
    },
    {
      path: '/student',
      name: 'student-home',
      component: () => import('../views/StudentView.vue')
    },
    {
      path: '/teacher',
      name: 'teacher-home',
      component: () => import('../views/TeacherView.vue')
    },
    {
      path: '/teacher/dashboard',
      name: 'teacher-dashboard',
      component: () => import('@/views/TeacherDashboard.vue')
    },
    {
      path: '/student/notes',
      name: 'student-notes',
      component: () => import('../views/StudentNotes.vue')
    }
  ],
})

// Global navigation guard.
//
// 1. If a token survived in localStorage but the user object is not loaded yet
//    (typical after a page reload), rehydrate the account from /api/me once.
// 2. Block protected routes for guests and bounce guest-only routes (login,
//    register) for users who are already signed in.
// router.beforeEach(async (to) => {
//   const auth = useAuthStore()

//   if (auth.token && !auth.user) {
//     await auth.fetchMe()
//   }

//   if (to.meta.requiresAuth && !auth.isAuthenticated) {
//     return { name: 'login', query: { redirect: to.fullPath } }
//   }

//   if (to.meta.guestOnly && auth.isAuthenticated) {
//     return { name: 'home' }
//   }
// })

export default router
