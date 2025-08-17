'use client';

import { Button } from '@/components/ui/button';
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from '@/components/ui/command';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Skeleton } from '@/components/ui/skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { ChevronDown, ChevronRight, Folder, FolderPlus, MoreHorizontal, Plus, Search, Settings2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';

interface Project {
    id: string;
    name: string;
    description?: string;
    key: string;
    created_at: string;
    updated_at: string;
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

interface OrganizationSidebarProps {
    projects: Project[];
    environments: Environment[];
    isLoading: boolean;
    selectedProjectId: string | null;
    selectedEnvKey: string | null;
    onProjectSelect: (project: Project) => void;
    onEnvironmentSelect: (env: Environment, projectId: string) => void;
    onCreateProject: (data: { name: string; slug: string; description?: string }) => Promise<void>;
    onCreateEnvironment: (data: { name: string; key: string; description?: string }, projectId: string) => Promise<void>;
}

export function OrganizationSidebar({
    projects,
    environments,
    isLoading,
    selectedProjectId,
    selectedEnvKey,
    onProjectSelect,
    onEnvironmentSelect,
    onCreateProject,
    onCreateEnvironment
}: OrganizationSidebarProps) {
    const [searchQuery, setSearchQuery] = useState('');
    const [expandedProjects, setExpandedProjects] = useState<Set<string>>(new Set());
    const [showCreateProject, setShowCreateProject] = useState(false);

    // Persist expanded state in localStorage
    useEffect(() => {
        const stored = localStorage.getItem('sidebar-expanded-projects');
        if (stored) {
            try {
                const expanded = JSON.parse(stored);
                setExpandedProjects(new Set(expanded));
            } catch {
                // Ignore errors
            }
        }
    }, []);

    useEffect(() => {
        localStorage.setItem('sidebar-expanded-projects', JSON.stringify(Array.from(expandedProjects)));
    }, [expandedProjects]);

    // Auto-expand selected project
    useEffect(() => {
        if (selectedProjectId && !expandedProjects.has(selectedProjectId)) {
            setExpandedProjects(prev => new Set(Array.from(prev).concat(selectedProjectId)));
        }
    }, [selectedProjectId, expandedProjects]);

    const toggleProjectExpanded = useCallback((projectId: string) => {
        setExpandedProjects(prev => {
            const newSet = new Set(Array.from(prev));
            if (newSet.has(projectId)) {
                newSet.delete(projectId);
            } else {
                newSet.add(projectId);
            }
            return newSet;
        });
    }, []);

    // Filter projects and environments based on search
    const filteredProjects = useMemo(() => {
        if (!searchQuery.trim()) return projects;

        const query = searchQuery.toLowerCase();
        return projects.filter(project =>
            project.name.toLowerCase().includes(query) ||
            project.key.toLowerCase().includes(query) ||
            environments.some(env =>
                env.name.toLowerCase().includes(query) ||
                env.key.toLowerCase().includes(query)
            )
        );
    }, [projects, environments, searchQuery]);

    const filteredEnvironments = useMemo(() => {
        if (!searchQuery.trim()) return environments;

        const query = searchQuery.toLowerCase();
        return environments.filter(env =>
            env.name.toLowerCase().includes(query) ||
            env.key.toLowerCase().includes(query)
        );
    }, [environments, searchQuery]);

    // Keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // "p" to create project
            if (e.key === 'p' && !e.ctrlKey && !e.metaKey && !e.altKey &&
                e.target instanceof HTMLElement &&
                !['INPUT', 'TEXTAREA'].includes(e.target.tagName)) {
                e.preventDefault();
                setShowCreateProject(true);
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, []);

    const handleCreateProject = async (name: string) => {
        const slug = name
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');

        await onCreateProject({ name, slug, description: '' });
        setShowCreateProject(false);
    };

    if (isLoading) {
        return (
            <div className="h-full flex flex-col">
                <div className="p-4 border-b">
                    <Skeleton className="h-9 w-full" />
                </div>
                <div className="flex-1 p-4 space-y-3">
                    {Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="space-y-2">
                            <Skeleton className="h-8 w-full" />
                            <div className="pl-4 space-y-1">
                                <Skeleton className="h-6 w-3/4" />
                                <Skeleton className="h-6 w-2/3" />
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col">
            {/* Header */}
            <div className="p-4 border-b space-y-3">
                <div className="flex items-center justify-between">
                    <h2 className="font-semibold text-sm">Projects</h2>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => setShowCreateProject(true)}
                                className="h-8 w-8 p-0"
                            >
                                <Plus className="h-4 w-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Create Project (P)</p>
                        </TooltipContent>
                    </Tooltip>
                </div>

                {/* Search */}
                <div className="relative">
                    <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Command className="rounded-lg border">
                        <CommandInput
                            placeholder="Search projects, environments..."
                            value={searchQuery}
                            onValueChange={setSearchQuery}
                            className="pl-8"
                        />
                    </Command>
                </div>
            </div>

            {/* Project Tree */}
            <ScrollArea className="flex-1">
                <div className="p-2">
                    {filteredProjects.length === 0 ? (
                        <div className="p-4 text-center">
                            <Folder className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                            <p className="text-sm text-muted-foreground">
                                {searchQuery ? 'No projects found' : 'No projects yet'}
                            </p>
                            {!searchQuery && (
                                <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setShowCreateProject(true)}
                                    className="mt-2 gap-2"
                                >
                                    <Plus className="h-4 w-4" />
                                    Create Project
                                </Button>
                            )}
                        </div>
                    ) : (
                        <div className="space-y-1">
                            {filteredProjects.map((project) => (
                                <div key={project.id}>
                                    {/* Project Row */}
                                    <div
                                        className={`group flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-accent cursor-pointer ${selectedProjectId === project.id ? 'bg-accent' : ''
                                            }`}
                                        onClick={() => onProjectSelect(project)}
                                        onContextMenu={(e) => {
                                            e.preventDefault();
                                            // TODO: Show context menu
                                        }}
                                    >
                                        <Button
                                            size="sm"
                                            variant="ghost"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                toggleProjectExpanded(project.id);
                                            }}
                                            className="h-4 w-4 p-0"
                                        >
                                            {expandedProjects.has(project.id) ? (
                                                <ChevronDown className="h-3 w-3" />
                                            ) : (
                                                <ChevronRight className="h-3 w-3" />
                                            )}
                                        </Button>

                                        <FolderPlus className="h-4 w-4 text-muted-foreground" />

                                        <div className="flex-1 min-w-0">
                                            <div className="text-sm font-medium truncate">
                                                {project.name}
                                            </div>
                                            <div className="text-xs text-muted-foreground truncate">
                                                {project.key}
                                            </div>
                                        </div>

                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button
                                                    size="sm"
                                                    variant="ghost"
                                                    className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
                                                    onClick={(e) => e.stopPropagation()}
                                                >
                                                    <MoreHorizontal className="h-3 w-3" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end">
                                                <DropdownMenuItem>Open</DropdownMenuItem>
                                                <DropdownMenuItem>Rename</DropdownMenuItem>
                                                <DropdownMenuItem>Archive</DropdownMenuItem>
                                            </DropdownMenuContent>
                                        </DropdownMenu>
                                    </div>

                                    {/* Environments */}
                                    {expandedProjects.has(project.id) && (
                                        <div className="ml-6 mt-1 space-y-0.5">
                                            {filteredEnvironments.map((env) => (
                                                <div
                                                    key={env.id}
                                                    className={`group flex items-center gap-2 px-2 py-1 rounded-md hover:bg-accent cursor-pointer text-sm ${selectedEnvKey === env.key ? 'bg-accent' : ''
                                                        }`}
                                                    onClick={() => onEnvironmentSelect(env, project.id)}
                                                >
                                                    <div className={`w-2 h-2 rounded-full ${env.enabled !== false ? 'bg-green-500' : 'bg-gray-400'
                                                        }`} />

                                                    <div className="flex-1 min-w-0">
                                                        <div className="font-medium truncate">
                                                            {env.name}
                                                        </div>
                                                        <div className="text-xs text-muted-foreground truncate">
                                                            {env.key}
                                                        </div>
                                                    </div>
                                                </div>
                                            ))}

                                            {/* Add Environment Button */}
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <Button
                                                        size="sm"
                                                        variant="ghost"
                                                        className="w-full justify-start gap-2 text-xs text-muted-foreground h-7"
                                                        onClick={() => {
                                                            // TODO: Open create environment modal
                                                        }}
                                                    >
                                                        <Plus className="h-3 w-3" />
                                                        Add Environment
                                                    </Button>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    <p>Create Environment (E)</p>
                                                </TooltipContent>
                                            </Tooltip>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </ScrollArea>

            {/* Create Project Modal - Simple inline for now */}
            {showCreateProject && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card p-6 rounded-lg w-96 border">
                        <h3 className="text-lg font-semibold mb-4">Create Project</h3>
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                const formData = new FormData(e.currentTarget);
                                const name = formData.get('name') as string;
                                if (name.trim()) {
                                    handleCreateProject(name.trim());
                                }
                            }}
                        >
                            <div className="space-y-4">
                                <div>
                                    <label className="text-sm font-medium">Project Name</label>
                                    <input
                                        name="name"
                                        type="text"
                                        className="w-full mt-1 px-3 py-2 border rounded-md"
                                        placeholder="Enter project name"
                                        autoFocus
                                        required
                                    />
                                </div>
                                <div className="flex justify-end gap-2">
                                    <Button
                                        type="button"
                                        variant="outline"
                                        onClick={() => setShowCreateProject(false)}
                                    >
                                        Cancel
                                    </Button>
                                    <Button type="submit">
                                        Create
                                    </Button>
                                </div>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}
