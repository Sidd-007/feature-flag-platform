# Feature Flag Platform - UI System Documentation

This document describes the modern, minimal, dark-only design system implemented for the Feature Flag Platform using shadcn/ui, Next.js App Router, and TypeScript.

## 🎨 Design System Overview

### Technology Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **UI Library**: shadcn/ui
- **Icons**: Lucide React
- **Animations**: tailwindcss-animate
- **Forms**: react-hook-form + zod validation
- **Tables**: @tanstack/react-table
- **Font**: DM Sans (Google Fonts)
- **Package Manager**: pnpm

### Theme & Colors

The platform uses a **dark-only** theme with a custom color palette optimized for accessibility and modern aesthetics:

```css
/* Primary color scheme */
--background: 0 0% 5%; /* Deep dark background */
--foreground: 0 0% 96%; /* Light text */
--muted: 0 0% 12%; /* Subtle backgrounds */
--muted-foreground: 0 0% 70%; /* Secondary text */

--card: 0 0% 7%; /* Card backgrounds */
--border: 0 0% 16%; /* Subtle borders */
--input: 0 0% 18%; /* Input backgrounds */

--primary: 221 83% 53%; /* Accent blue */
--secondary: 0 0% 14%; /* Secondary surfaces */
--destructive: 0 84% 60%; /* Error red */

--radius: 0.75rem; /* Border radius */
```

### Visual Language

- **Monochrome surfaces** with a single accent color (primary blue)
- **Subtle 1px borders** with soft shadows
- **Generous spacing** using 8/12/16/24px increments
- **Rounded corners** (0.75rem default radius)
- **Typography hierarchy** with DM Sans font family

## 🧩 Component Architecture

### Primitive Components (`/components/primitives/`)

Reusable, low-level components that form the foundation of the UI:

#### `PageHeader`

```tsx
<PageHeader title="Feature Flag Platform" description="Optional description">
  <Button>Action</Button>
</PageHeader>
```

#### `Section`

```tsx
<Section
  title="Projects"
  description="Manage your projects"
  headerAction={<Button>Create</Button>}
>
  {/* Content */}
</Section>
```

#### `StatusDot`

```tsx
<StatusDot variant="success">Connected to localhost:8080</StatusDot>
<StatusDot variant="error">Connection failed</StatusDot>
```

#### `CopyButton`

```tsx
<CopyButton text="token-to-copy" size="icon" variant="ghost" />
```

#### `EmptyState`

```tsx
<EmptyState
  icon={Building2}
  title="No organizations yet"
  description="Create your first organization to get started."
  action={{
    label: "Create Organization",
    onClick: handleCreate,
  }}
/>
```

#### `Kbd`

```tsx
<Kbd>⌘K</Kbd> {/* Keyboard shortcut display */}
```

### shadcn/ui Components

All standard UI components from shadcn/ui are available and styled consistently:

- `Button` - Primary, secondary, outline, destructive variants
- `Input` - Standard height (h-9) with proper styling
- `Textarea` - Consistent styling with min-height controls
- `Card` - Rounded corners with subtle shadows
- `Badge` - Type indicators and status labels
- `Table` - Data tables with hover states
- `Dialog` - Modal dialogs for forms and confirmations
- `Sheet` - Slide-over panels for detailed views
- `Form` - React Hook Form integration with validation
- `Switch` - Toggle controls with optimistic updates
- `Select` - Dropdown selectors
- `Skeleton` - Loading states
- `Toast` - Notifications and feedback

## 📄 Page Structure

### Organizations Dashboard (`/`)

**Layout**: 3-column responsive grid

- **Create Organization** (xl:col-span-2) - Form with zod validation
- **API Status** - Connection indicator with retry button
- **Your Organizations** (xl:col-span-3) - Data table with actions

### Organization Management (`/organizations/[orgId]`)

**Layout**: 12-column responsive grid

- **Projects** (xl:col-span-4) - Project creation and selection
- **Environments** (xl:col-span-4) - Environment management
- **Feature Flags** (xl:col-span-12) - Comprehensive flag management

#### Feature Flag Management

- **Filters**: Search, type, and status filtering
- **Data Table**: Name, key, type, version, state, actions
- **Create Dialog**: Multi-step flag creation
- **Actions**: Toggle state, publish/unpublish, edit, delete

## 🎯 Interaction Patterns

### Form Handling

- **Validation**: Real-time with zod schemas
- **Error States**: Inline error messages
- **Loading States**: Disabled buttons with loading text
- **Success Feedback**: Toast notifications

### Data Loading

- **Skeleton States**: Show loading placeholders
- **Empty States**: Helpful illustrations and CTAs
- **Error Handling**: Clear error messages with retry options

### Optimistic Updates

- **Toggle Switches**: Immediate visual feedback
- **Publishing**: Show loading state with revert on error
- **Form Submissions**: Disable and show progress

### Micro-interactions

- **Hover States**: Subtle background changes
- **Focus States**: Clear focus rings for accessibility
- **Animations**: Smooth transitions with tailwindcss-animate
- **Copy Actions**: Visual feedback and toast confirmation

## 🛠 Component Usage Guidelines

### Button Variants

```tsx
// Primary actions
<Button variant="default">Create</Button>

// Secondary actions
<Button variant="secondary">Cancel</Button>

// Quiet actions
<Button variant="outline">Manage</Button>

// Destructive actions
<Button variant="destructive">Delete</Button>
```

### Badge Usage

```tsx
// Type indicators
<Badge variant="default">boolean</Badge>

// Version indicators
<Badge variant="outline">Published v8</Badge>

// State indicators
<Badge variant="secondary">ON</Badge>
```

### Input Styling

```tsx
// Standard inputs with consistent height
<Input className="h-9" placeholder="Enter value..." />

// Textareas with minimum height
<Textarea className="min-h-[80px]" placeholder="Description..." />
```

### Card Layouts

```tsx
<Card className="rounded-2xl border bg-card p-5 md:p-6 shadow-sm">
  <CardHeader className="p-0 pb-5">
    <CardTitle className="text-lg font-semibold">Title</CardTitle>
    <CardDescription>Description</CardDescription>
  </CardHeader>
  <CardContent className="p-0">{/* Content */}</CardContent>
</Card>
```

## ♿ Accessibility Features

### ARIA Support

- **Labels**: All interactive elements have proper labels
- **Descriptions**: Screen reader friendly descriptions
- **Live Regions**: Status updates announced to screen readers

### Keyboard Navigation

- **Focus Management**: Logical tab order
- **Shortcuts**: Support for common keyboard shortcuts
- **Escape Handling**: Close dialogs and sheets with Escape

### Visual Accessibility

- **Focus Indicators**: Clear focus rings on all interactive elements
- **Color Independence**: Never rely on color alone for meaning
- **Text Contrast**: High contrast ratios for readability

## 🚀 Performance Optimizations

### Code Splitting

- **Dynamic Imports**: Large components loaded on demand
- **Route-based Splitting**: Automatic with Next.js App Router

### Loading Strategies

- **Skeleton Screens**: Immediate visual feedback
- **Optimistic Updates**: Instant UI updates with error rollback
- **Pagination**: Limit data loading for large tables

### Bundle Optimization

- **Tree Shaking**: Unused components automatically removed
- **Font Loading**: Optimized with next/font/google
- **CSS Optimization**: Tailwind purges unused styles

## 📱 Responsive Design

### Breakpoints

```tsx
// Mobile-first approach
className = "grid grid-cols-1 xl:grid-cols-3 gap-6";

// Responsive spacing
className = "p-5 md:p-6";

// Conditional visibility
className = "hidden xl:block";
```

### Mobile Considerations

- **Touch Targets**: Minimum 44px tap areas
- **Horizontal Scrolling**: Tables scroll horizontally on mobile
- **Sheet Sizing**: Responsive sheet widths

## 🔧 Development Workflow

### Adding New Components

1. **Create Component**: Use TypeScript with proper interfaces
2. **Style with Tailwind**: Follow design system tokens
3. **Add to Primitives**: Export from `/components/primitives/index.ts`
4. **Document Props**: Include JSDoc comments
5. **Test Accessibility**: Verify keyboard and screen reader support

### Form Creation Pattern

```tsx
const schema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
});

type FormData = z.infer<typeof schema>;

const form = useForm<FormData>({
  resolver: zodResolver(schema),
  defaultValues: { name: "", description: "" },
});

const onSubmit = async (data: FormData) => {
  // Handle submission
};

// In JSX
<Form {...form}>
  <form onSubmit={form.handleSubmit(onSubmit)}>
    <FormField
      control={form.control}
      name="name"
      render={({ field }) => (
        <FormItem>
          <FormLabel>Name</FormLabel>
          <FormControl>
            <Input {...field} className="h-9" />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  </form>
</Form>;
```

### API Integration Pattern

```tsx
const [data, setData] = useState<Type[]>([]);
const [loading, setLoading] = useState(true);

const loadData = async () => {
  setLoading(true);
  try {
    const response = await apiClient.getData();
    if (response.error) {
      toast({
        title: "Error",
        description: response.error,
        variant: "destructive",
      });
    } else {
      // Handle response data normalization
      let responseData = response.data;
      if (
        responseData &&
        typeof responseData === "object" &&
        !Array.isArray(responseData)
      ) {
        responseData = responseData.data || responseData.items || responseData;
      }
      setData(Array.isArray(responseData) ? responseData : []);
    }
  } catch (err) {
    toast({
      title: "Error",
      description: "An unexpected error occurred",
      variant: "destructive",
    });
  } finally {
    setLoading(false);
  }
};
```

## 🎯 Best Practices

### Component Design

- **Single Responsibility**: Each component has one clear purpose
- **Composability**: Components work well together
- **Consistency**: Follow established patterns
- **Accessibility**: Always consider screen readers and keyboard users

### Styling

- **Design Tokens**: Use CSS custom properties
- **Utility Classes**: Prefer Tailwind utilities over custom CSS
- **Responsive Design**: Mobile-first approach
- **Dark Theme**: All components work in dark mode

### State Management

- **Local State**: Use useState for component-specific state
- **Form State**: Use react-hook-form for forms
- **Server State**: Handle loading, error, and success states
- **Optimistic Updates**: Provide immediate feedback

### Error Handling

- **User-Friendly Messages**: Clear, actionable error messages
- **Graceful Degradation**: Handle API failures gracefully
- **Retry Mechanisms**: Provide retry options where appropriate
- **Loading States**: Always show loading feedback

---

This UI system provides a solid foundation for building modern, accessible, and maintainable user interfaces. All components follow consistent patterns and can be easily extended or customized as needed.
