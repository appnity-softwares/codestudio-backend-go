# MVP API Contract

## Overview
This document outlines the final, standardized API surface for the MVP.  
**Base URL**: `/api`

---

## 1. Authentication Module
**Usage**: Login, Signup, and Global Auth State.

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Register a new user | `/register` |
| `POST` | `/auth/login` | Login user (returns token) | `/login` |
| `POST` | `/auth/logout` | Invalidate session (client-side) | `Navbar` (Logout Button) |
| `GET` | `/auth/google/login` | Initiate Google OAuth | `/login` (Social Auth) |
| `GET` | `/auth/github/login` | Initiate GitHub OAuth | `/login` (Social Auth) |
| `POST` | `/auth/forgot-password`| Request password reset | `/forgot-password` |
| `POST` | `/auth/reset-password` | Reset password with token | `/reset-password` |

---

## 2. Snippets Module
**Usage**: Core Code Sharing & Execution.

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `GET` | `/snippets` | List snippets (Search/Filter) | `/snippets` (Feed), Home |
| `GET` | `/snippets/:id` | Get snippet details | `/snippets/[id]` |
| `POST` | `/snippets/:id/run` | **Execute Snippet Code** (No input) | `/snippets/[id]` (Run Button) |
| `POST` | `/snippets` | Create new snippet | `/snippets/new` |
| `PUT` | `/snippets/:id` | Update snippet | `/snippets/[id]/edit` |
| `DELETE` | `/snippets/:id` | Delete snippet | `/snippets/[id]`, Dashboard |
| `PATCH` | `/snippets/:id/output`| Approve/Update Output Snapshot | `/snippets/[id]` (Author Only) |

---

## 3. User & Profile Module
**Usage**: User Profiles and Dashboard.

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `GET` | `/users` | List users (Community) | `/community` |
| `GET` | `/users/:username` | Public Profile | `/profile/[username]` |
| `GET` | `/users/:username/snippets`| Get public snippets of user | `/profile/[username]` |
| `GET` | `/users/profile` | **My Profile** (Settings) | `/settings/profile` |
| `PUT` | `/users/profile` | Update Profile | `/settings/profile` |
| `GET` | `/users/profile/stats` | **Dashboard Stats** (MVP Stats) | `/dashboard` |

---

## 4. Messaging Module (1-1)
**Usage**: Private Messaging.

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `GET` | `/chat/contacts` | List chat contacts | `/chat` (Sidebar) |
| `GET` | `/chat/messages` | Get message history | `/chat` (Conversation) |
| `POST` | `/chat/read/:senderId` | Mark messages as read | `/chat` (On Open) |

---

## 5. Contests (Events & Arena) Module
**Usage**: Competitive Programming Events.

### Events & Registration
| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `GET` | `/events` | List Events | `/events` |
| `GET` | `/events/:id` | Event Details | `/events/[id]` |
| `POST` | `/events/:id/register` | Register for Event | `/events/[id]` |
| `GET` | `/events/:id/access` | Check Access (Guard) | `/arena/[id]` (Guard) |
| `GET` | `/registrations/my` | User's Registrations | `/dashboard` |

### Arena (Problems & Submission)
| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `GET` | `/contests/:id/problems` | List Problems | `/arena/[id]` |
| `GET` | `/contests/:id/leaderboard`| Event Leaderboard | `/arena/[id]/leaderboard` |
| `GET` | `/contests/:id/problems/:pid`| Problem Details | `/arena/[id]/problem/[pid]` |
| `POST` | `/contests/:id/problems/:pid/submit`| **Submit Solution** | `/arena/[id]/problem/[pid]` |
| `POST` | `/contests/:id/problems/:pid/run`| **Run Sample Test** | `/arena/[id]/problem/[pid]` |
| `GET` | `/contests/:id/problems/:pid/submissions`| Submission History | `/arena/[id]/problem/[pid]` |

### Admin (Events)
| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `POST` | `/events` | Create Event | Admin Panel |
| `POST` | `/contests/:id/problems` | Create Problem | Admin Panel |
| `PUT` | `/contests/:id/problems/:pid`| Update Problem | Admin Panel |
| `DELETE` | `/contests/:id/problems/:pid`| Delete Problem | Admin Panel |
| `GET` | `/registrations` | List All Registrations | Admin Panel |
| `PATCH` | `/registrations/:id/status`| Update Reg Status | Admin Panel |

---

## 6. Payments Module
**Usage**: Payment Processing (Events).

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `POST` | `/payments/order` | Create Razorpay Order | Payment Modal |
| `POST` | `/payments/verify` | Verify Payment | Payment Modal |

---

## 7. Upload Module
**Usage**: File and Image Uploads.

| Method | Endpoint | Description | Frontend Page / Component |
| :--- | :--- | :--- | :--- |
| `POST` | `/upload/profile-image` | Upload Avatar | `/settings/profile` |
| `POST` | `/upload/chat-attachment`| Send Chat Image/File | `/chat` |
| `POST` | `/upload` | Generic Upload | General |

---

## Notes
*   **Removed Features**: Social graph (follows), snippet likes/comments/saves, activity feed, global leaderboard, and complex gamification.
*   **Execution**: All code execution (`/run`) is standardized to use the internal Piston service with fixed limits.
