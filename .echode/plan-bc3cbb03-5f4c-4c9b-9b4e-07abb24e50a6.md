# Mobile Responsiveness Implementation

## Overview
The goal is to enhance the mobile responsiveness of the Admin Dashboard. The current implementation relies on tables that do not scale well to small screens. We will implement a "mobile-first" approach by introducing a card-based layout for mobile devices while retaining the table layout for desktop. We will also refine global layout paddings and modal widths to ensuring a polished experience on all devices.

## Principles
- **Mobile-First**: Design for small screens first, then enhance for larger ones.
- **High Standards**: Use modular components (splitting `KeyCard` vs `KeyRow`) to keep code clean (DRY/SOLID).
- **Responsive Utilities**: Leverage Tailwind's breakpoint prefixes (`sm:`, `md:`, `lg:`) effectively.

## Proposed Changes

### 1. Component: `KeyList` (Refactor)
**File**: `frontend/src/components/admin/KeyList.tsx`
- **Current State**: Uses a standard `Table` which forces horizontal scrolling on mobile.
- **New Strategy**:
  - Extract the current row logic into a shared helper or keep distinct `KeyRow` (table) and `KeyCard` (mobile) components.
  - **New Component**: `KeyCard` (internal or separate file if large).
    - Displays Key info, Note, Rate Limit, and Actions in a vertical stacked card layout.
    - Uses `Card` component from UI library.
  - **Rendering Logic**:
    - Mobile (`< md`): Show stack of `KeyCard` components.
    - Desktop (`>= md`): Show the existing `Table`.
    - Use Tailwind classes `md:hidden` and `hidden md:block` for switching.

### 2. Component: `Dashboard` (Layout Tweaks)
**File**: `frontend/src/components/admin/Dashboard.tsx`
- **Current State**: `px-6` padding on mobile might be slightly too wide for very small screens (320px), or conversely, we want to maximize space.
- **Change**: Adjust main container padding to `px-4 sm:px-8` to align with mobile best practices (16px edge spacing).

### 3. Component: `Modal` (UI Library)
**File**: `frontend/src/components/ui/modal.tsx`
- **Current State**: `w-full max-w-lg`. On screens smaller than `max-w-lg`, it might touch the edges.
- **Change**: Add `w-[95%] sm:w-full` or `mx-4` to ensure it maintains a safety margin on mobile devices while staying centered.

### 4. Component: `EditKeyModal` (Review)
**File**: `frontend/src/components/admin/EditKeyModal.tsx`
- **Action**: Verify internal inputs (especially `ModelSelector`) flow correctly within the new modal width constraints. (Likely no code change needed if `Modal` is fixed, but we will verify).

## Implementation Steps

1.  **Update UI Primitives**: 
    - Modify `Modal` in `frontend/src/components/ui/modal.tsx` to include mobile width safety margins.
2.  **Adjust Layout**:
    - Update `Dashboard` in `frontend/src/components/admin/Dashboard.tsx` for responsive padding.
3.  **Implement Mobile Key List**:
    - Edit `frontend/src/components/admin/KeyList.tsx`.
    - Create `KeyCard` component for mobile view.
    - Implement the responsive switch (`hidden`/`block` classes).
    - Ensure all actions (Copy, Edit, Revoke) work identically in both views.