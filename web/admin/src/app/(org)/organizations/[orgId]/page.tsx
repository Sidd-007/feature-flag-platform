'use client';

import { CreateProjectDialog } from '@/components/org/CreateProjectDialog';
import { ProjectCard } from '@/components/org/ProjectCard';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { useToast } from '@/hooks/use-toast';
import { useSelection } from '@/hooks/useSelection';
import apiClient from '@/lib/api';
import { Building2, Plus } from 'lucide-react';
import { useParams } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

interface Project {
    id: string;
    name: string;
    description?: string;
    key: string;
    created_at: string;
    updated_at: string;
}

export default function OrganizationHomePage() {
    const params = useParams();
    const orgId = params.orgId as string;
    const { toast } = useToast();
    const selection = useSelection();

    // State
    const [projects, setProjects] = useState<Project[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [showCreateDialog, setShowCreateDialog] = useState(false);

    // Load projects
    const loadProjects = useCallback(async () => {
        setIsLoading(true);
        try {
            const response = await apiClient.getProjects(orgId);
            if (response.error) {
                toast({
                    title: "Error loading projects",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                let projectsData = response.data;
                if (projectsData && typeof projectsData === 'object' && !Array.isArray(projectsData)) {
                    const dataObj = projectsData as any;
                    projectsData = dataObj.data || dataObj.projects || dataObj.items || projectsData;
                }
                const projectsArray = Array.isArray(projectsData) ? projectsData : [];
                setProjects(projectsArray);
            }
        } catch (err) {
            toast({
                title: "Failed to load projects",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        } finally {
            setIsLoading(false);
        }
    }, [orgId, toast]);

    useEffect(() => {
        loadProjects();
    }, [loadProjects]);

    // Handle ?goto=last query param
    useEffect(() => {
        const searchParams = new URLSearchParams(window.location.search);
        if (searchParams.get('goto') === 'last' && projects.length > 0) {
            const lastSelection = selection.getLastSelection(orgId);
            if (lastSelection?.projectId && lastSelection?.envKey) {
                // Navigate to the last selection
                selection.navigateToFlags(lastSelection.projectId, lastSelection.envKey);
                return;
            }
        }
    }, [projects, orgId, selection]);

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
                await loadProjects();

                // Navigate to the new project
                const newProject = response.data as Project;
                if (newProject?.id) {
                    selection.navigateToProject(newProject.id);
                }
            }
        } catch (err) {
            toast({
                title: "Failed to create project",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    if (isLoading) {
        return (
            <div className="p-8">
                <div className="mb-8">
                    <Skeleton className="h-8 w-64 mb-2" />
                    <Skeleton className="h-4 w-96" />
                </div>

                <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                    {Array.from({ length: 6 }).map((_, i) => (
                        <Skeleton key={i} className="h-48 rounded-2xl" />
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div className="p-8">
            {/* Header */}
            <div className="mb-8">
                <h1 className="text-2xl font-bold mb-2">Projects</h1>
                <p className="text-muted-foreground">
                    Manage your feature flag projects and environments
                </p>
            </div>

            {/* Content */}
            {projects.length === 0 ? (
                // Empty State
                <div className="flex flex-col items-center justify-center py-16">
                    <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4">
                        <Building2 className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <h3 className="text-lg font-semibold mb-2">Create your first project</h3>
                    <p className="text-muted-foreground mb-6 text-center max-w-md">
                        Projects help you organize your feature flags by application, team, or environment.
                        Get started by creating your first project.
                    </p>
                    <Button
                        onClick={() => setShowCreateDialog(true)}
                        className="gap-2"
                    >
                        <Plus className="h-4 w-4" />
                        Create Project
                    </Button>
                </div>
            ) : (
                // Projects Grid
                <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                    {projects.map((project) => (
                        <ProjectCard
                            key={project.id}
                            project={project}
                            onOpen={(project: Project) => selection.navigateToProject(project.id)}
                            onRename={() => {
                                // TODO: Implement rename
                                toast({
                                    title: "Coming soon",
                                    description: "Project renaming will be implemented soon.",
                                });
                            }}
                            onArchive={() => {
                                // TODO: Implement archive
                                toast({
                                    title: "Coming soon",
                                    description: "Project archiving will be implemented soon.",
                                });
                            }}
                        />
                    ))}
                </div>
            )}

            {/* Floating Action Button */}
            {projects.length > 0 && (
                <Button
                    className="fixed bottom-8 right-8 rounded-full h-14 w-14 shadow-lg"
                    onClick={() => setShowCreateDialog(true)}
                >
                    <Plus className="h-6 w-6" />
                </Button>
            )}

            {/* Create Project Dialog */}
            <CreateProjectDialog
                open={showCreateDialog}
                onOpenChange={setShowCreateDialog}
                onSubmit={createProject}
            />
        </div>
    );
}
