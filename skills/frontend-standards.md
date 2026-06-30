# Frontend Standards — OpsNexus React Applications

Standards for all React/TypeScript applications in `apps/`. Read this before writing a component.

---

## 1. Project Structure

All frontend apps use feature-based organization. A "feature" is a self-contained slice of the application (authentication, case management, user administration, etc.).

```
apps/{name}/
  src/
    features/           # Feature modules — the primary location for app code
      auth/
        components/     # Components used only within this feature
        hooks/          # React hooks for this feature
        pages/          # Route-level page components
        services/       # API call functions for this feature
        types.ts        # TypeScript types specific to this feature
        index.ts        # Re-exports for use from outside the feature
      cases/
        ...
    components/         # Shared, reusable components used across features
      ui/               # Pure presentational components (Button, Input, Modal)
      layout/           # Layout components (Sidebar, Header, PageWrapper)
    services/
      api.ts            # Shared API client (single axios instance)
    hooks/              # Shared hooks
    types/              # Global TypeScript types and API response types
    lib/                # Utilities and helpers
    App.tsx
    main.tsx
  public/
  index.html
  vite.config.ts
  tsconfig.json
```

If a component is only used in one feature, it belongs in that feature's `components/`. If it's used in two or more features, it moves to `src/components/`.

---

## 2. Component Design

### Use function components

All components are function components. Class components are not used.

```tsx
// CORRECT
export function CaseCard({ caseData, onSelect }: CaseCardProps) {
  return (
    <div onClick={() => onSelect(caseData.id)}>
      <h3>{caseData.title}</h3>
    </div>
  );
}

// WRONG
class CaseCard extends React.Component<CaseCardProps> { ... }
```

### Props via TypeScript interfaces

Every component prop is typed. `any` is not a valid prop type.

```tsx
// CORRECT
interface CaseCardProps {
  caseData: Case;
  onSelect: (id: string) => void;
  isSelected?: boolean;
}

// WRONG
function CaseCard(props: any) { ... }
function CaseCard({ caseData, onSelect }: { caseData: any; onSelect: any }) { ... }
```

### Keep components small

If a component exceeds ~100 lines, it should be split. Common extraction points:
- A visual sub-section that could stand alone → extract to a child component
- Logic that requires 3+ hooks or complex derived state → extract to a custom hook
- A list item rendered in a `.map()` → extract to its own component

### Conditional class names

Use `clsx` (or the project's `cn` utility) for conditional classes. Never concatenate class strings.

```tsx
// CORRECT
import { cn } from '@/lib/utils';

<button className={cn(
  'px-4 py-2 rounded font-medium',
  isActive && 'bg-blue-600 text-white',
  isDisabled && 'opacity-50 cursor-not-allowed',
  size === 'sm' && 'text-sm px-2 py-1',
)}>

// WRONG
<button className={`px-4 py-2 ${isActive ? 'bg-blue-600 text-white' : ''} ${isDisabled ? 'opacity-50' : ''}`}>
```

---

## 3. Data Fetching

### Always use TanStack Query

Server state (data that comes from the API) is managed with TanStack Query. This is not optional.

```tsx
// CORRECT
import { useQuery } from '@tanstack/react-query';
import { casesApi } from '../services/casesApi';

function CasesList() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['cases', tenantId, filters],
    queryFn: () => casesApi.list({ tenantId, ...filters }),
  });

  if (isLoading) return <CasesListSkeleton />;
  if (isError) return <ErrorMessage error={error} />;
  if (!data?.length) return <EmptyState message="No cases found" />;

  return <div>{data.map(c => <CaseCard key={c.id} caseData={c} />)}</div>;
}

// WRONG: useEffect + fetch for server data
function CasesList() {
  const [cases, setCases] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch('/api/v1/cases').then(r => r.json()).then(setCases).finally(() => setLoading(false));
  }, []);
}
```

### Handle all states

Every data-driven component handles three states: loading, error, empty (no data), and populated. Skipping any of these is a bug.

### Mutations and cache invalidation

```tsx
const queryClient = useQueryClient();

const createCase = useMutation({
  mutationFn: casesApi.create,
  onSuccess: () => {
    // Invalidate so the list refetches with the new case
    queryClient.invalidateQueries({ queryKey: ['cases'] });
    toast.success('Case created successfully');
  },
  onError: (error) => {
    toast.error(getErrorMessage(error));
  },
});
```

---

## 4. Forms

All forms use `react-hook-form` with `zod` validation. Uncontrolled inputs without `react-hook-form` are not permitted.

```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const createCaseSchema = z.object({
  title: z.string().min(3, 'Title must be at least 3 characters').max(200),
  priority: z.enum(['low', 'medium', 'high', 'critical']),
  description: z.string().max(5000).optional(),
});

type CreateCaseFormData = z.infer<typeof createCaseSchema>;

function CreateCaseForm({ onSuccess }: { onSuccess: () => void }) {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<CreateCaseFormData>({
    resolver: zodResolver(createCaseSchema),
  });

  const onSubmit = async (data: CreateCaseFormData) => {
    await createCase.mutateAsync(data);
    onSuccess();
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <label htmlFor="title">Title</label>
      <input id="title" {...register('title')} aria-describedby="title-error" />
      {errors.title && <span id="title-error" role="alert">{errors.title.message}</span>}
      {/* ... */}
      <button type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Creating...' : 'Create Case'}
      </button>
    </form>
  );
}
```

The Zod schema is the single source of truth for validation rules. Server-side validation errors (422 responses) must also be surfaced to the user — use `form.setError()` for that.

---

## 5. State Management

| State Type | Solution |
|-----------|---------|
| Server data (API responses) | TanStack Query |
| Auth state (current user, token) | React Context |
| Local UI state (open/closed, active tab) | `useState` |
| Cross-feature shared state | React Context (sparingly) |
| Complex derived state | `useMemo` / `useReducer` |

Do not reach for a global store (Zustand, Redux) unless you have a concrete problem that Context and Query cannot solve. That problem has not arisen in this project; document the justification if it does.

---

## 6. TypeScript Rules

- `any` is banned. ESLint will flag it. Use `unknown` and narrow the type.
- All API response shapes are defined in `src/types/api.ts` or `features/{name}/types.ts`
- Use `unknown` in catch blocks and narrow before use:

```tsx
// CORRECT
try {
  await api.createCase(data);
} catch (error: unknown) {
  if (error instanceof AxiosError && error.response?.data?.error) {
    toast.error(error.response.data.error.message);
  } else {
    toast.error('An unexpected error occurred');
  }
}

// WRONG
} catch (error: any) {
  toast.error(error.message); // crashes if error doesn't have .message
}
```

- All event handler parameter types must be explicit:
```tsx
// CORRECT
const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => { ... }

// WRONG
const handleChange = (event) => { ... }
```

---

## 7. API Client

One axios instance for the entire application. Located at `src/services/api.ts`. Never create a second one.

```ts
// src/services/api.ts
import axios from 'axios';

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
});

// Attach auth token from localStorage/context on every request
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  const tenantId = getTenantId();
  if (tenantId) {
    config.headers['X-Tenant-ID'] = tenantId;
  }
  return config;
});

// Handle 401 globally: redirect to login
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      clearAuth();
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

Feature-level API functions import `apiClient` and add their type annotations:

```ts
// features/cases/services/casesApi.ts
import { apiClient } from '@/services/api';
import { Case, CreateCaseInput, PaginatedResponse } from '@/types/api';

export const casesApi = {
  list: (params: CaseListParams): Promise<PaginatedResponse<Case>> =>
    apiClient.get('/api/v1/cases', { params }).then(r => r.data),

  getById: (id: string): Promise<Case> =>
    apiClient.get(`/api/v1/cases/${id}`).then(r => r.data.data),

  create: (input: CreateCaseInput): Promise<Case> =>
    apiClient.post('/api/v1/cases', input).then(r => r.data.data),
};
```

---

## 8. Error Handling

API errors must be caught and shown as user-friendly messages. Raw error objects or internal error strings must never be displayed to users.

```tsx
// CORRECT
const errorMessage = getErrorMessage(error); // extracts .error.message from API envelope

function getErrorMessage(error: unknown): string {
  if (error instanceof AxiosError) {
    return error.response?.data?.error?.message ?? 'An unexpected error occurred';
  }
  return 'An unexpected error occurred';
}

// WRONG: exposes raw error
<p>{error.message}</p>
<p>{JSON.stringify(error)}</p>
```

Log the full error object (including stack) to the console in development. In production, send to your error tracking service.

---

## 9. Accessibility

Accessibility is a requirement, not an afterthought.

- Use semantic HTML: `<button>` for buttons (not `<div onClick>`), `<nav>`, `<main>`, `<section>`, `<article>`
- Every `<input>`, `<select>`, `<textarea>` has a `<label>` with a matching `htmlFor`, or `aria-label`
- Interactive elements that look like buttons but are not `<button>` need `role="button"` and keyboard handlers
- Color alone must not convey meaning (status indicators need text or icon + text)
- Focus is visible for keyboard navigation (do not remove `outline` without a replacement)
- Images have `alt` text; decorative images use `alt=""`
- Error messages are associated with their input via `aria-describedby`

---

## 10. Testing

Use Vitest + Testing Library for unit/component tests. Playwright for E2E.

```tsx
// features/cases/components/CaseCard.test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CaseCard } from './CaseCard';

const mockCase: Case = {
  id: 'case-1',
  title: 'Server is down',
  priority: 'critical',
  status: 'open',
};

test('renders case title and priority', () => {
  render(<CaseCard caseData={mockCase} onSelect={vi.fn()} />);

  expect(screen.getByText('Server is down')).toBeInTheDocument();
  expect(screen.getByText('Critical')).toBeInTheDocument();
});

test('calls onSelect with case id when clicked', async () => {
  const onSelect = vi.fn();
  render(<CaseCard caseData={mockCase} onSelect={onSelect} />);

  await userEvent.click(screen.getByRole('article'));

  expect(onSelect).toHaveBeenCalledWith('case-1');
});
```

Rules:
- Query by role, label, or visible text — not by test IDs or class names
- Use MSW to intercept API calls in tests that involve `useQuery`
- Render components inside a test wrapper that provides `QueryClientProvider`, `BrowserRouter`, and `AuthContext`
- Test the three states: loading, error, and data

```tsx
// test/utils.tsx — shared render wrapper
export function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          {ui}
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
```
