'use client';

import { useParams } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { FlagsDataTable } from '@/components/organization/FlagsDataTable';
import { FlagsToolbar } from '@/components/organization/FlagsToolbar';
import { useToast } from '@/hooks/use-toast';
import { useOptimizedData } from '@/hooks/useOptimizedData';
import { useSelection } from '@/hooks/useSelection';
import apiClient from '@/lib/api';

interface Flag {
    id: string;
    key: string;
    name: string;
    description?: string;
    type: string;
    enabled: boolean;
    default_value: any;
    created_at: string;
    published?: boolean;
    version?: number;
    isPublishing?: boolean;
}

interface Environment {
    id: string;
    name: string;
    key: string;
    description?: string;
    created_at: string;
    enabled?: boolean;
    isDefault?: boolean;
}

export default function FlagsPage() {
    const params = useParams();
    const orgId = params.orgId as string;
    const projectId = params.projectId as string;
    const envKey = params.envKey as string;
    const { toast } = useToast();
    const selection = useSelection();

    // Optimized data loading for environment
    const {
        data: environment,
        isLoading: environmentLoading,
        error: environmentError,
    } = useOptimizedData({
        key: `environment:${orgId}:${projectId}:${envKey}`,
        fetchFn: async () => {
            // First get all environments for the project
            const response = await apiClient.getEnvironments(orgId, projectId);
            if (response.error) {
                toast({
                    title: "Error loading environment",
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
            const environments = Array.isArray(envData) ? envData : [];

            // Find the specific environment by key
            const foundEnv = environments.find((env: any) => env.key === envKey);
            if (!foundEnv) {
                throw new Error('Environment not found');
            }

            return foundEnv as Environment;
        },
        dependencies: [orgId, projectId, envKey],
        onError: (err) => {
            toast({
                title: "Failed to load environment",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        },
    });

    // Optimized data loading for flags
    const {
        data: flags,
        isLoading: flagsLoading,
        refresh: refreshFlags,
    } = useOptimizedData({
        key: `flags:${orgId}:${projectId}:${environment?.id || 'none'}`,
        fetchFn: async () => {
            if (!environment) return [];

            const response = await apiClient.getFlags(orgId, projectId, environment.id);
            if (response.error) {
                toast({
                    title: "Error loading flags",
                    description: response.error,
                    variant: "destructive",
                });
                throw new Error(response.error);
            }

            let flagsData = response.data;
            if (flagsData && typeof flagsData === 'object' && !Array.isArray(flagsData)) {
                const dataObj = flagsData as any;
                flagsData = dataObj.data || dataObj.flags || dataObj.items || flagsData;
            }
            const rawArray = Array.isArray(flagsData) ? flagsData : [];
            const normalized: Flag[] = rawArray.map((f: any) => ({
                id: f.id,
                key: f.key,
                name: f.name,
                description: f.description,
                type: f.type || 'boolean',
                enabled: f.status ? String(f.status).toLowerCase() === 'active' : !!f.enabled,
                default_value: f.default_variation ?? f.default_value,
                created_at: f.created_at,
                published: f.published,
                version: f.version,
                isPublishing: false,
            }));
            return normalized;
        },
        dependencies: [orgId, projectId, environment?.id],
        immediate: !!environment,
        onError: (err) => {
            toast({
                title: "Failed to load flags",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        },
    });

    // Update selection when environment loads - fixed dependency array
    useEffect(() => {
        if (environment) {
            selection.setProjectId(projectId);
            selection.setEnvKey(envKey);
        }
    }, [environment, projectId, envKey, selection.setProjectId, selection.setEnvKey]);

    const handleToggleFlag = async (flag: Flag) => {
        if (!environment) return;

        try {
            const response = await apiClient.updateFlag(orgId, projectId, environment.id, flag.key, {
                enabled: !flag.enabled,
            });
            if (response.error) {
                toast({
                    title: "Failed to toggle flag",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                toast({
                    title: "Success",
                    description: `Flag "${flag.name}" ${flag.enabled ? 'disabled' : 'enabled'}.`,
                });
                await refreshFlags();
            }
        } catch (err) {
            toast({
                title: "Failed to toggle flag",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const handlePublishFlag = async (flag: Flag) => {
        if (!environment) return;

        try {
            if (flag.published) {
                // Unpublish flag
                const response = await apiClient.unpublishFlag(orgId, projectId, environment.id, flag.key);
                if (response.error) {
                    toast({
                        title: "Failed to unpublish flag",
                        description: response.error,
                        variant: "destructive",
                    });
                } else {
                    toast({
                        title: "Success",
                        description: `Flag "${flag.name}" unpublished successfully.`,
                    });
                    await refreshFlags();
                }
            } else {
                // Publish flag
                const response = await apiClient.publishFlag(orgId, projectId, environment.id, flag.key);
                if (response.error) {
                    toast({
                        title: "Failed to publish flag",
                        description: response.error,
                        variant: "destructive",
                    });
                } else {
                    toast({
                        title: "Success",
                        description: `Flag "${flag.name}" published successfully.`,
                    });
                    await refreshFlags();
                }
            }
        } catch (err) {
            toast({
                title: "Failed to update flag",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const isLoading = environmentLoading || flagsLoading;

    if (isLoading) {
        return (
            <div className="p-8">
                <div className="mb-8">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground mb-4">
                        <div className="h-4 w-16 bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                        <span>/</span>
                        <div className="h-4 w-24 bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                        <span>/</span>
                        <div className="h-4 w-20 bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                    </div>
                    <div className="h-8 w-48 bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse mb-2" />
                    <div className="h-4 w-96 bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                </div>

                <div className="space-y-4">
                    <div className="h-12 w-full bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                    <div className="h-96 w-full bg-zinc-300 dark:bg-zinc-700 rounded animate-pulse" />
                </div>
            </div>
        );
    }

    if (environmentError || !environment) {
        return (
            <div className="p-8">
                <div className="text-center py-16">
                    <h2 className="text-xl font-semibold mb-2">Environment not found</h2>
                    <p className="text-muted-foreground mb-4">
                        The environment you're looking for doesn't exist or you don't have access to it.
                    </p>
                </div>
            </div>
        );
    }

    return (
        <div className="p-8">
            {/* Breadcrumb */}
            <div className="flex items-center gap-2 text-sm text-muted-foreground mb-4">
                <button
                    onClick={() => selection.navigateToOrgHome()}
                    className="hover:text-foreground transition-colors"
                >
                    Projects
                </button>
                <span>/</span>
                <button
                    onClick={() => selection.navigateToProject(projectId)}
                    className="hover:text-foreground transition-colors"
                >
                    {projectId}
                </button>
                <span>/</span>
                <span>{envKey}</span>
            </div>

            {/* Header */}
            <div className="mb-8">
                <h1 className="text-3xl font-bold mb-2">Feature Flags</h1>
                <p className="text-muted-foreground text-lg">
                    Manage feature flags for {projectId} in {environment.name}
                </p>
            </div>

            {/* Toolbar */}
            <FlagsToolbar
                searchQuery={selection.q}
                onSearchChange={selection.setQuery}
                typeFilter={selection.type}
                onTypeChange={selection.setType}
                statusFilter={selection.status}
                onStatusChange={selection.setStatus}
                selectedEnvKey={envKey}
                environments={[environment]}
                onEnvChange={(env: string) => selection.setEnvKey(env)}
                onCreateFlag={() => {
                    // TODO: Implement create flag
                    toast({
                        title: "Coming soon",
                        description: "Flag creation will be implemented soon.",
                    });
                }}
                canCreateFlag={true}
            />

            {/* Data Table */}
            <div className="mt-6">
                <FlagsDataTable
                    flags={flags || []}
                    isLoading={flagsLoading}
                    currentPage={selection.page}
                    pageSize={selection.pageSize}
                    totalFlags={flags?.length || 0}
                    onPageChange={selection.setPage}
                    onPageSizeChange={selection.setPageSize}
                    onToggleFlag={handleToggleFlag}
                    onPublishFlag={handlePublishFlag}
                    onEditFlag={() => {
                        // TODO: Implement edit flag
                        toast({
                            title: "Coming soon",
                            description: "Flag editing will be implemented soon.",
                        });
                    }}
                    onDuplicateFlag={() => {
                        // TODO: Implement duplicate flag
                        toast({
                            title: "Coming soon",
                            description: "Flag duplication will be implemented soon.",
                        });
                    }}
                    onDeleteFlag={() => {
                        // TODO: Implement delete flag
                        toast({
                            title: "Coming soon",
                            description: "Flag deletion will be implemented soon.",
                        });
                    }}
                />
            </div>
        </div>
    );
}
