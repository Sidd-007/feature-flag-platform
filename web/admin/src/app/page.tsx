'use client';

import { EmptyState, PageHeader, Section, StatusDot } from '@/components/primitives';
import { CopyButton } from '@/components/primitives/CopyButton';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { useToast } from '@/hooks/use-toast';
import apiClient from '@/lib/api';
import { zodResolver } from '@hookform/resolvers/zod';
import { Building2, LogOut, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

interface Organization {
    id: string;
    name: string;
    description?: string;
    slug: string;
    billing_tier: string;
    created_at: string;
    updated_at: string;
}

const createOrgSchema = z.object({
    name: z.string().min(1, 'Organization name is required'),
    description: z.string().optional(),
});

type CreateOrgData = z.infer<typeof createOrgSchema>;

export default function Dashboard() {
    const router = useRouter();
    const { toast } = useToast();
    const [organizations, setOrganizations] = useState<Organization[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [apiConnected, setApiConnected] = useState(true);

    const form = useForm<CreateOrgData>({
        resolver: zodResolver(createOrgSchema),
        defaultValues: {
            name: '',
            description: '',
        },
    });

    useEffect(() => {
        // Check if user is authenticated
        const token = localStorage.getItem('auth_token');
        if (!token) {
            router.push('/login');
            return;
        }

        loadOrganizations();
    }, [router]);

    const loadOrganizations = async () => {
        setIsLoading(true);
        try {
            const response = await apiClient.getOrganizations();

            if (response.error) {
                if (response.status === 401) {
                    router.push('/login');
                    return;
                }
                setApiConnected(false);
                toast({
                    title: "Error loading organizations",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                setApiConnected(true);
                // Handle the actual API response structure
                let orgData = response.data;

                if (orgData && typeof orgData === 'object' && !Array.isArray(orgData)) {
                    if ((orgData as any).data) {
                        orgData = (orgData as any).data;
                    } else if ((orgData as any).organizations) {
                        orgData = (orgData as any).organizations;
                    } else if ((orgData as any).items) {
                        orgData = (orgData as any).items;
                    }
                }

                const organizations = Array.isArray(orgData) ? orgData : [];
                setOrganizations(organizations);
            }
        } catch (err) {
            console.error('Error loading organizations:', err);
            setApiConnected(false);
            toast({
                title: "Failed to load organizations",
                description: "Could not connect to the API",
                variant: "destructive",
            });
        } finally {
            setIsLoading(false);
        }
    };

    const onSubmit = async (data: CreateOrgData) => {
        try {
            const response = await apiClient.createOrganization({
                name: data.name,
                description: data.description,
            });

            if (response.error) {
                toast({
                    title: "Failed to create organization",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                toast({
                    title: "Success",
                    description: `Organization "${data.name}" created successfully.`,
                });
                form.reset();
                await loadOrganizations();
            }
        } catch (err) {
            toast({
                title: "Failed to create organization",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const handleLogout = () => {
        apiClient.logout();
        router.push('/login');
    };

    const openOrganization = (orgId: string) => {
        router.push(`/organizations/${orgId}`);
    };

    if (isLoading) {
        return (
            <div className="min-h-screen bg-background p-6">
                <div className="mx-auto max-w-7xl">
                    <div className="flex items-center justify-between py-8">
                        <div className="space-y-2">
                            <Skeleton className="h-8 w-64" />
                            <Skeleton className="h-4 w-96" />
                        </div>
                        <Skeleton className="h-10 w-20" />
                    </div>
                    <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
                        <div className="xl:col-span-2">
                            <Skeleton className="h-96 w-full" />
                        </div>
                        <div>
                            <Skeleton className="h-32 w-full" />
                        </div>
                        <div className="xl:col-span-3">
                            <Skeleton className="h-64 w-full" />
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background">
            <div className="mx-auto max-w-7xl p-6">
                <PageHeader title="Feature Flag Platform">
                    <Button variant="outline" onClick={handleLogout} className="gap-2">
                        <LogOut className="h-4 w-4" />
                        Logout
                    </Button>
                </PageHeader>

                <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
                    {/* Create Organization */}
                    <div className="xl:col-span-2">
                        <Section
                            title="Create New Organization"
                            description="Start by creating an organization to manage your feature flags"
                        >
                            <Form {...form}>
                                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                                    <FormField
                                        control={form.control}
                                        name="name"
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel>Organization Name</FormLabel>
                                                <FormControl>
                                                    <Input
                                                        placeholder="My Company"
                                                        {...field}
                                                        className="h-9"
                                                    />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                    <FormField
                                        control={form.control}
                                        name="description"
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel>Description (Optional)</FormLabel>
                                                <FormControl>
                                                    <Textarea
                                                        placeholder="Description of your organization"
                                                        {...field}
                                                        className="min-h-[80px]"
                                                    />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                    <Button
                                        type="submit"
                                        className="w-full gap-2"
                                        disabled={form.formState.isSubmitting}
                                    >
                                        <Plus className="h-4 w-4" />
                                        {form.formState.isSubmitting ? 'Creating...' : 'Create'}
                                    </Button>
                                </form>
                            </Form>
                        </Section>
                    </div>

                    {/* API Status */}
                    <div>
                        <Section title="API Status">
                            <StatusDot
                                variant={apiConnected ? "success" : "error"}
                            >
                                {apiConnected ? "Connected to localhost:8080" : "Disconnected from API"}
                            </StatusDot>
                            {!apiConnected && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={loadOrganizations}
                                    className="mt-4"
                                >
                                    Retry Connection
                                </Button>
                            )}
                        </Section>
                    </div>

                    {/* Your Organizations */}
                    <div className="xl:col-span-3">
                        <Section title="Your Organizations" description="Organizations you have access to">
                            {organizations.length === 0 ? (
                                <EmptyState
                                    icon={Building2}
                                    title="No organizations yet"
                                    description="Create your first organization above to get started with feature flags."
                                />
                            ) : (
                                <div className="rounded-md border">
                                    <Table>
                                        <TableHeader>
                                            <TableRow>
                                                <TableHead>Name</TableHead>
                                                <TableHead>ID</TableHead>
                                                <TableHead>Created</TableHead>
                                                <TableHead className="text-right">Actions</TableHead>
                                            </TableRow>
                                        </TableHeader>
                                        <TableBody>
                                            {organizations.map((org) => (
                                                <TableRow
                                                    key={org.id}
                                                    className="hover:bg-muted/50"
                                                >
                                                    <TableCell>
                                                        <div>
                                                            <div className="font-medium">{org.name}</div>
                                                            {org.description && (
                                                                <div className="text-sm text-muted-foreground line-clamp-2">
                                                                    {org.description}
                                                                </div>
                                                            )}
                                                        </div>
                                                    </TableCell>
                                                    <TableCell>
                                                        <div className="flex items-center gap-2">
                                                            <code className="text-xs bg-muted px-2 py-1 rounded">
                                                                {org.id}
                                                            </code>
                                                            <CopyButton text={org.id} />
                                                        </div>
                                                    </TableCell>
                                                    <TableCell>
                                                        <div className="text-sm text-muted-foreground">
                                                            {new Date(org.created_at).toLocaleDateString()}
                                                        </div>
                                                    </TableCell>
                                                    <TableCell className="text-right">
                                                        <div className="flex items-center justify-end gap-2">
                                                            <Button
                                                                onClick={() => openOrganization(org.id)}
                                                                size="sm"
                                                            >
                                                                Open
                                                            </Button>
                                                            <DropdownMenu>
                                                                <DropdownMenuTrigger asChild>
                                                                    <Button variant="ghost" size="sm">
                                                                        ⋯
                                                                    </Button>
                                                                </DropdownMenuTrigger>
                                                                <DropdownMenuContent align="end">
                                                                    <DropdownMenuItem onClick={() => openOrganization(org.id)}>
                                                                        Open
                                                                    </DropdownMenuItem>
                                                                    <DropdownMenuItem>
                                                                        Rename
                                                                    </DropdownMenuItem>
                                                                    <DropdownMenuItem className="text-destructive">
                                                                        Delete
                                                                    </DropdownMenuItem>
                                                                </DropdownMenuContent>
                                                            </DropdownMenu>
                                                        </div>
                                                    </TableCell>
                                                </TableRow>
                                            ))}
                                        </TableBody>
                                    </Table>
                                </div>
                            )}
                        </Section>
                    </div>
                </div>
            </div>
        </div>
    );
}