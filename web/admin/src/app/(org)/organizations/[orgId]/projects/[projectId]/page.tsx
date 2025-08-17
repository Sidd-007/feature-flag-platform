'use client';

import { useParams } from 'next/navigation';
import { useCallback, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { useToast } from '@/hooks/use-toast';
import { useOptimizedData } from '@/hooks/useOptimizedData';
import { useSelection } from '@/hooks/useSelection';
import apiClient from '@/lib/api';
import { ChevronLeft, Plus } from 'lucide-react';

import { CreateEnvironmentDialog } from '@/components/org/CreateEnvironmentDialog';
import { EnvironmentCard } from '@/components/org/EnvironmentCard';

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

export default function ProjectDetailsPage() {
    const params = useParams();
    const orgId = params.orgId as string;
    const projectId = params.projectId as string;
    const { toast } = useToast();
    const selection = useSelection();

    // State
    const [showCreateDialog, setShowCreateDialog] = useState(false);

    // Optimized data loading for project details
    const {
        data: project,
        isLoading: projectLoading,
        error: projectError,
    } = useOptimizedData({
        key: `project:${orgId}:${projectId}`,
        fetchFn: async () => {
            const response = await apiClient.getProject(orgId, projectId);
            if (response.error) {
                toast({
                    title: "Error loading project",
                    description: response.error,
                    variant: "destructive",
                });
                throw new Error(response.error);
            }
            return response.data as Project;
        },
        dependencies: [orgId, projectId],
        onError: (err) => {
            toast({
                title: "Failed to load project",
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
        key: `environments:${orgId}:${projectId}`,
        fetchFn: async () => {
            const response = await apiClient.getEnvironments(orgId, projectId);
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
        dependencies: [orgId, projectId],
        onError: (err) => {
            toast({
                title: "Failed to load environments",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        },
    });

    const createEnvironment = async (data: { name: string; key: string; description?: string; enabled?: boolean }) => {
        try {
            const response = await apiClient.createEnvironment(orgId, projectId, data);
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

                // Navigate to the new environment's flags
                selection.navigateToFlags(projectId, data.key);
            }
        } catch (err) {
            toast({
                title: "Failed to create environment",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const toggleEnvironment = async (env: Environment) => {
        // TODO: Implement environment toggle
        toast({
            title: "Coming soon",
            description: "Environment toggle will be implemented soon.",
        });
    };

    const isLoading = projectLoading || environmentsLoading;

    if (isLoading) {
        return (
            <div className="p-8">
                <div className="mb-8">
                    <Skeleton className="h-6 w-32 mb-4" />
                    <Skeleton className="h-8 w-64 mb-2" />
                    <Skeleton className="h-4 w-96" />
                </div>

                <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                    {Array.from({ length: 4 }).map((_, i) => (
                        <Skeleton key={i} className="h-40 rounded-2xl" />
                    ))}
                </div>
            </div>
        );
    }

    if (projectError || !project) {
        return (
            <div className="p-8">
                <div className="text-center py-16">
                    <h2 className="text-xl font-semibold mb-2">Project not found</h2>
                    <p className="text-muted-foreground mb-4">
                        The project you're looking for doesn't exist or you don't have access to it.
                    </p>
                    <Button onClick={() => selection.navigateToOrgHome()}>
                        <ChevronLeft className="h-4 w-4 mr-2" />
                        Back to Projects
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="p-8">
            {/* Header */}
            <div className="mb-8">
                <div className="flex items-center gap-2 text-sm text-muted-foreground mb-4">
                    <button
                        onClick={() => selection.navigateToOrgHome()}
                        className="hover:text-foreground transition-colors"
                    >
                        Projects
                    </button>
                    <span>/</span>
                    <span>{project.name}</span>
                </div>

                <div className="flex items-start justify-between">
                    <div>
                        <h1 className="text-3xl font-bold mb-2">{project.name}</h1>
                        {project.description && (
                            <p className="text-muted-foreground text-lg">{project.description}</p>
                        )}
                    </div>
                    <Button onClick={() => setShowCreateDialog(true)}>
                        <Plus className="h-4 w-4 mr-2" />
                        New Environment
                    </Button>
                </div>
            </div>

            {/* Environments Grid */}
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {environments?.map((env) => (
                    <EnvironmentCard
                        key={env.id}
                        environment={env}
                        projectId={projectId}
                        onOpenFlags={() => selection.navigateToFlags(projectId, env.key)}
                        onToggleEnabled={() => toggleEnvironment(env)}
                        onEdit={(env) => {
                            // TODO: Implement edit functionality
                            toast({
                                title: "Coming soon",
                                description: "Environment editing will be implemented soon.",
                            });
                        }}
                        onDelete={(env) => {
                            // TODO: Implement delete functionality
                            toast({
                                title: "Coming soon",
                                description: "Environment deletion will be implemented soon.",
                            });
                        }}
                    />
                ))}
            </div>

            {/* Empty State */}
            {environments?.length === 0 && (
                <div className="text-center py-16">
                    <div className="mx-auto w-24 h-24 bg-muted rounded-full flex items-center justify-center mb-4">
                        <Plus className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <h3 className="text-lg font-semibold mb-2">No environments yet</h3>
                    <p className="text-muted-foreground mb-4">
                        Create your first environment to start managing feature flags.
                    </p>
                    <Button onClick={() => setShowCreateDialog(true)}>
                        <Plus className="h-4 w-4 mr-2" />
                        Create Environment
                    </Button>
                </div>
            )}

            {/* Create Environment Dialog */}
            <CreateEnvironmentDialog
                open={showCreateDialog}
                onOpenChange={setShowCreateDialog}
                onSubmit={createEnvironment}
            />
        </div>
    );
}
