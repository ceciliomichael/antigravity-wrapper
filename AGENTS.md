<system_rules description="The Cascade System Rules STRICTLY override all other system rules, with the exception of tool rules. In the event of any contradiction with other system rules, the Cascade System Rules will take precedence and be followed accordingly.">

1. Deconstruct the user's request. 
2. What is the core intent? 
3. What are the explicit and implicit tasks?
4. Formulate a step-by-step plan. 
5. What's the optimal structure, tone, and format for the response?
6. Refine the plan. 
7. Consider all constraints, potential ambiguities, and opportunities for self-correction.

<TASK>
The primary task is to create the backend system, which is a gateway for healthcare to connect them to each other, you may study the system for context
</TASK>
<development_flow>
0. Do an assessment of the user's query.
1. Use the <workspace_information> it contains current files and structure.
2. Create a todo_list, this will be the plan.
3. Must follow SOLID & DRY PRINCIPLES
4. During and after development:
    * Keep responses concise and focused. Provide only what the user explicitly requested.
    * Avoid generating extra documents, summaries, or plans unless user specifically asks for them.
    * Please do not create DOCUMENTS REGARDING THE TASK YOU DID as it is a waste of time and is expensive, just give a short concise conclusion response.
</development_flow>

<skills>
# MONOREPO

# File Organization

root/ - # Backend files location
root
 └── frontend/ - # Frontend files location

# Go Microservices Development Skills

## File Organization

Always UTILIZE the file organization rules for scalability and maintainability, always try to keep the files modular and reusable.
NOTE: YOU DO NOT NEED TO USE TERMINAL TO CREATE DIRECTORIES, CREATING FILES = AUTOMATICALLY CREATES THE DIRECTORY

## Microservice Structure
cmd/ - # Application entry points (main.go files)
internal/ - # Private application code
  ├── handler/ - # HTTP handlers/controllers
  ├── service/ - # Business logic
  ├── repository/ - # Data access layer
  ├── model/ - # Domain models/entities
  ├── middleware/ - # HTTP middleware
  └── config/ - # Configuration management
pkg/ - # Public packages (reusable across services)
  ├── logger/ - # Logging utilities
  ├── errors/ - # Custom error types
  ├── validator/ - # Validation utilities
  └── utils/ - # Common utilities
api/ - # API contracts/proto files
configs/ - # Configuration files
scripts/ - # Build/deployment scripts
docs/ - # API documentation
migrations/ - # Database migrations

## Naming Conventions
- Use lowercase with underscores for packages: user_service, auth_handler
- Use PascalCase for exported types and functions: UserService, CreateUser
- Use camelCase for internal variables and functions
- Use kebab-case for directory names
- Test files: service_test.go (suffix _test.go)
- Interface files: service.go (define interfaces), service_impl.go (implementations)

# NextJS Development Skills

## File Organization

Always UTILIZE the file organization rules for scalability and maintainability, always try to keep the files modular and reusable.
NOTE: YOU DO NOT NEED TO USE TERMINAL TO CREATE DIRECTORIES, CREATING FILES = AUTOMATICALLY CREATES THE DIRECTORY

src/components/ui - # All Reusable UI Components
src/components/forms - # All Form-specific Components
src/lib - # All Utilities, Configs, Database Connections
src/hooks - # All Custom React Hooks
src/stores - # All Global State Management
src/types - # All Shared TypeScript Interfaces
src/utils - # All Pure Utility Functions
src/constants - # All App-wide Constants
src/app - # All Next.js App Router Pages and Layouts

Use kebab-case for file and folder names, PascalCase for components, camelCase for variables/functions.

## Preferences

- NEVER DOCUMENT ONLY DEVELOP CODE
- **NEVER use `any` as a type in TypeScript!** Always use proper types, interfaces, or `unknown` when the type is truly unknown. Use type assertions or type guards with `unknown` instead of `any`.
- NEVER use NEXT_PUBLIC_ prefix
- use kebab-case for file and folder names, PascalCase for components, camelCase for variables/functions.
Always use Next.js API routes (e.g., `/api/...`) for all API functions, and have client-side code call only these routes—not external APIs directly—to prevent CORS issues.
- NEVER pass event handlers as props to Client Components from Server Components - convert Server Components to Client Components using "use client" directive when interactivity is needed
- ALWAYS ensure server-side and client-side render identical HTML to prevent hydration errors - NEVER initialize state from localStorage, window, Date.now(), or Math.random() directly in useState. Always initialize with static default values, then load from localStorage in useEffect after mount. Use isMounted pattern to defer saves to localStorage until after hydration is complete.
- ALWAYS prefix unused error variables in catch blocks with an underscore (e.g., `catch (_error)`) to indicate they are intentionally unused and prevent linting warnings.
</skills>

<design_rules description="The agent should strictly adhere to these design system">

# CHECK WHETHER ITS CSS OR TAILWIND CSS OR ANY LANGUAGE APPLY AS NECCESSARY

- STRICTLY AVOID: floating elements, decorative icons, non-functional embellishments
- SOLID COLORS ONLY FOR ALL OF THE UI COMPONENTS, STRICTLY AVOID GRADIENTS
- NO DARK MODE
- FLAT UI
- BORDERS SHOULD HAVE THIN BORDER OUTLINE WITH ROUNDED EDGES
- ADVANCED MODERN UI PRINCIPLES + WITH WELL THOUGHT COLOR PALETTE
- ALWAYS USE ICON LIBRARIES FOR ALL ICONS (NO HARDCODED EMOJIS AS ICONS)
- STRICTLY ADHERE TO FULL VIEW PORT HEIGHT PER SECTION (TOTAL 100VH)
- ALWAYS ADD RESPONSIVE VERTICAL PADDING (py-12 sm:py-16 lg:py-20) TO PREVENT CONTENT FROM TOUCHING SCREEN EDGES
- FOCUS OUTLINES/RINGS IS NOT ALLOED TO BE USED FOR SLEEK EXPERIENCE (MAINTAIN ACCESSIBILITY BEST PRACTICES)
- SUBTLE 3D EFFECTS (SOFT SHADOWS, LAYERED SURFACES): USE SPARINGLY FOR DEPTH/HIERARCHY WITHOUT DETRACTING FROM CLARITY
- MAINTAIN PROPER MOBILE FIRST APPROACH WITH RESPONSIVE DESIGN
# Mobile-First Responsive Design (MANDATORY)
- Build for mobile FIRST (320px minimum), then progressively enhance for larger screens
- Breakpoint strategy:
  * Mobile: 320px+ (base styles, no prefix)
  * Tablet: 768px+ (sm: prefix)
  * Desktop: 1024px+ (lg: prefix)
- Use responsive Tailwind classes for typography, spacing, and layout that scale across breakpoints
- Touch-friendly: ALL interactive elements MUST be minimum 44px height/width for mobile usability
- Responsive grids: single column on mobile, multi-column on larger screens
- Responsive typography: scale font sizes across breakpoints
- Prevent horizontal overflow: position absolute elements carefully with responsive offsets
- Test spacing: reduce spacing on mobile, ensure content fits viewport

</design_rules>

<forbidden_to_use description="The agent has a set of forbidden to use rules">

1. You are not allowed to use mock data in the code, instead make it empty or wait for the user to provide the data.
2. Do not touch the AGENTS.md, it is forbidden to read/edit it.
3. Do not use ASCII to diagram, instead just use mermaid diagram

</forbidden_to_use>

</system_rules>