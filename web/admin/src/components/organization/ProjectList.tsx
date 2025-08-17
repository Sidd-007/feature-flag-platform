'use client';

import { CopyButton } from '@/components/primitives';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Archive, Edit, ExternalLink, FolderPlus, MoreHorizontal } from 'lucide-react';
import { memo } from 'react';
import { EmptyState } from './EmptyState';

interface Project {
    id: string;
    name: string;
    description?: string;
    key: string;
    created_at: string;
    updated_at: string;
}

interface ProjectListProps {
    projects: Project[];
    selectedProjectId?: string | null;
    onSelectProject: (project: Project) => void;
    onEditProject?: (project: Project) => void;
    onArchiveProject?: (project: Project) => void;
}

const ProjectItem = memo(({
    project,
    isSelected,
    onSelect,
    onEdit,
    onArchive
}: {
    project: Project;
    isSelected: boolean;
    onSelect: () => void;
    onEdit?: () => void;
    onArchive?: () => void;
}) => {
    return (
        <div
            className={`group relative p-4 border rounded-lg cursor-pointer transition-all duration-200 ${isSelected
                ? 'bg-muted/50 border-muted-foreground/20 shadow-sm ring-1 ring-muted-foreground/20'
                : 'hover:bg-muted/30 hover:border-muted-foreground/10'
                }`}
            onClick={onSelect}
        >
            <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-sm line-clamp-1">{project.name}</h3>
                    <div className="flex items-center gap-2 mt-2">
                        <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                            {project.key}
                        </code>
                        <CopyButton text={project.key} />
                    </div>
                    {project.description && (
                        <p className="text-xs text-muted-foreground mt-2 line-clamp-2">
                            {project.description}
                        </p>
                    )}
                    <div className="text-xs text-muted-foreground mt-3">
                        Updated {new Date(project.updated_at).toLocaleDateString()}
                    </div>
                </div>

                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                            onClick={(e) => e.stopPropagation()}
                        >
                            <MoreHorizontal className="h-4 w-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onSelect(); }}>
                            <ExternalLink className="h-4 w-4 mr-2" />
                            Open
                        </DropdownMenuItem>
                        {onEdit && (
                            <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onEdit(); }}>
                                <Edit className="h-4 w-4 mr-2" />
                                Rename
                            </DropdownMenuItem>
                        )}
                        {onArchive && (
                            <DropdownMenuItem
                                onClick={(e) => { e.stopPropagation(); onArchive(); }}
                                className="text-destructive"
                            >
                                <Archive className="h-4 w-4 mr-2" />
                                Archive
                            </DropdownMenuItem>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
            </div>
        </div>
    );
});

ProjectItem.displayName = 'ProjectItem';

export function ProjectList({
    projects,
    selectedProjectId,
    onSelectProject,
    onEditProject,
    onArchiveProject
}: ProjectListProps) {
    if (projects.length === 0) {
        return (
            <EmptyState
                icon={FolderPlus}
                title="No projects yet"
                description="Create your first project above to get started."
            />
        );
    }

    return (
        <div className="space-y-3">
            {projects.map((project) => (
                <ProjectItem
                    key={project.id}
                    project={project}
                    isSelected={selectedProjectId === project.id}
                    onSelect={() => onSelectProject(project)}
                    onEdit={onEditProject ? () => onEditProject(project) : undefined}
                    onArchive={onArchiveProject ? () => onArchiveProject(project) : undefined}
                />
            ))}
        </div>
    );
}
