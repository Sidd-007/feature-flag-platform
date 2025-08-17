'use client';

import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { ApiTokensSheet } from '@/components/ApiTokensSheet';
import { OrganizationSidebar } from '@/components/org/Sidebar';
import { useToast } from '@/hooks/use-toast';
import { useOptimizedData } from '@/hooks/useOptimizedData';
import { useSelection } from '@/hooks/useSelection';
import apiClient from '@/lib/api';

interface Project {
    id: string;
    name: string;
    slug: string;
    description?: string;
    created_at: string;
    updated_at: string;
}

interface Environment {
    id: string;
    name: string;
    key: string;
    description?: string;
    created_at: string;
    updated_at: string;
    enabled?: boolean;
    isDefault?: boolean;
}

interface OrganizationLayoutProps {
    children: React.ReactNode;
}

export default function OrganizationLayout({ children }: OrganizationLayoutProps) {
    const router = useRouter();
    const params = useParams();
    const orgId = params.orgId as string;
    const { toast } = useToast();
    const selection = useSelection();

    // State
    const [showTokensSheet, setShowTokensSheet] = useState(false);

    // Optimized data loading for projects
    const {
        data: projects,
        isLoading: projectsLoading,
        refresh: refreshProjects,
    } = useOptimizedData({
        key: `projects:${orgId}`,
        fetchFn: async () => {
            const response = await apiClient.getProjects(orgId);
            if (response.error) {
                toast({
                    title: "Error loading projects",
                    description: response.error,
                    variant: "destructive",
                });
                throw new Error(response.error);
            }

            let projectsData = response.data;
            if (projectsData && typeof projectsData === 'object' && !Array.isArray(projectsData)) {
                const dataObj = projectsData as any;
                projectsData = dataObj.data || dataObj.projects || dataObj.items || projectsData;
            }
            return Array.isArray(projectsData) ? projectsData : [];
        },
        dependencies: [orgId],
        onError: (err) => {
            toast({
                title: "Failed to load projects",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        },
    });

    // Optimized data loading for environments
    const {
        data: environments,
        isLoading: environmentsLoading,
        refresh: refreshEnvironments,
    } = useOptimizedData({
        key: `environments:${orgId}:${selection.projectId || 'none'}`,
        fetchFn: async () => {
            if (!selection.projectId) return [];

            const response = await apiClient.getEnvironments(orgId, selection.projectId);
            if (response.error) {
                toast({
                    title: "Error loading environments",
                    description: response.error,
                    variant: "destructive",
                });
                throw new Error(response.error);
            }

            let envData = response.data;
            if (envData && typeof envData === 'object' && !Array.isArray(envData)) {
                const dataObj = envData as any;
                envData = dataObj.data || dataObj.environments || dataObj.items || envData;
            }
            return Array.isArray(envData) ? envData : [];
        },
        dependencies: [orgId, selection.projectId],
        immediate: !!selection.projectId,
        onError: (err) => {
            toast({
                title: "Failed to load environments",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        },
    });

    const createProject = async (data: { name: string; slug: string; description?: string }) => {
        try {
            const response = await apiClient.createProject(orgId, {
                name: data.name,
                slug: data.slug,
                description: data.description,
            });

            if (response.error) {
                toast({
                    title: "Failed to create project",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                toast({
                    title: "Success",
                    description: `Project "${data.name}" created successfully.`,
                });
                await refreshProjects();
            }
        } catch (err) {
            toast({
                title: "Failed to create project",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const createEnvironment = async (data: { name: string; key: string; description?: string; enabled?: boolean }) => {
        if (!selection.projectId) return;

        try {
            const response = await apiClient.createEnvironment(orgId, selection.projectId, data);
            if (response.error) {
                toast({
                    title: "Failed to create environment",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                toast({
                    title: "Success",
                    description: `Environment "${data.name}" created successfully.`,
                });
                await refreshEnvironments();
            }
        } catch (err) {
            toast({
                title: "Failed to create environment",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    // Check authentication on mount
    useEffect(() => {
        const token = localStorage.getItem('auth_token');
        if (!token) {
            router.push('/login');
        }
    }, [router]);

    return (
        <div className="flex h-screen bg-background">
            {/* Sidebar */}
            <div className="w-80 border-r border-border bg-card">
                <OrganizationSidebar
                    projects={projects || []}
                    environments={environments || []}
                    isLoading={projectsLoading}
                    selectedProjectId={selection.projectId}
                    selectedEnvKey={selection.envKey}
                    onProjectSelect={(project) => selection.navigateToProject(project.id)}
                    onEnvironmentSelect={(env, projectId) => selection.navigateToFlags(projectId, env.key)}
                    onCreateProject={createProject}
                    onCreateEnvironment={createEnvironment}
                />
            </div>

            {/* Main content */}
            <div className="flex-1 flex flex-col overflow-hidden">
                {/* Header */}
                <header className="h-16 border-b border-border bg-card px-6 flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                        <span className="text-sm font-mono text-muted-foreground">
                            {orgId}
                        </span>
                    </div>
                    <div className="flex items-center space-x-2">
                        <button
                            onClick={() => setShowTokensSheet(true)}
                            className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input hover:bg-accent hover:text-accent-foreground h-9 px-3"
                        >
                            <svg
                                className="h-4 w-4 mr-2"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                                xmlns="http://www.w3.org/2000/svg"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                                />
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                                />
                            </svg>
                            Manage API Tokens
                        </button>
                        <button
                            onClick={() => selection.navigateToOrgHome()}
                            className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input hover:bg-accent hover:text-accent-foreground h-9 px-3"
                        >
                            <svg
                                className="h-4 w-4 mr-2"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                                xmlns="http://www.w3.org/2000/svg"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M10 19l-7-7m0 0l7-7m-7 7h18"
                                />
                            </svg>
                            ← Back to Organizations
                        </button>
                    </div>
                </header>

                {/* Page content */}
                <main className="flex-1 overflow-auto">
                    {children}
                </main>
            </div>

            {/* API Tokens Sheet */}
            <ApiTokensSheet
                open={showTokensSheet}
                onOpenChange={setShowTokensSheet}
                orgId={orgId}
                projectId={selection.projectId || undefined}
                envId={selection.envKey || undefined}
            />
        </div>
    );
}
