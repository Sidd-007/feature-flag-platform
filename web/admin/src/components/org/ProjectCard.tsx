'use client';

import { CopyButton } from '@/components/primitives';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Archive, Edit, ExternalLink, FolderOpen, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';

interface Project {
    id: string;
    name: string;
    description?: string;
    key: string;
    created_at: string;
    updated_at: string;
}

interface ProjectCardProps {
    project: Project;
    onOpen: (project: Project) => void;
    onRename: (project: Project) => void;
    onArchive: (project: Project) => void;
}

export function ProjectCard({ project, onOpen, onRename, onArchive }: ProjectCardProps) {
    const [isContextMenuOpen, setIsContextMenuOpen] = useState(false);

    const handleContextMenu = (e: React.MouseEvent) => {
        e.preventDefault();
        setIsContextMenuOpen(true);
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric'
        });
    };

    return (
        <Card
            className="group hover:shadow-md transition-all duration-200 hover:border-primary/20 cursor-pointer rounded-2xl border"
            onClick={() => onOpen(project)}
            onContextMenu={handleContextMenu}
        >
            <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                        <h3 className="font-semibold text-lg mb-1 truncate">
                            {project.name}
                        </h3>
                        {project.description && (
                            <p className="text-sm text-muted-foreground line-clamp-2 mb-3">
                                {project.description}
                            </p>
                        )}

                        {/* Project Key with Copy Button */}
                        <div className="flex items-center gap-2">
                            <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                                {project.key}
                            </code>
                            <CopyButton text={project.key} />
                        </div>
                    </div>

                    <DropdownMenu open={isContextMenuOpen} onOpenChange={setIsContextMenuOpen}>
                        <DropdownMenuTrigger asChild>
                            <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    setIsContextMenuOpen(true);
                                }}
                            >
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onOpen(project);
                                }}
                                className="gap-2"
                            >
                                <ExternalLink className="h-4 w-4" />
                                Open
                            </DropdownMenuItem>
                            <DropdownMenuItem
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onRename(project);
                                }}
                                className="gap-2"
                            >
                                <Edit className="h-4 w-4" />
                                Rename
                            </DropdownMenuItem>
                            <DropdownMenuItem
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onArchive(project);
                                }}
                                className="gap-2 text-destructive focus:text-destructive"
                            >
                                <Archive className="h-4 w-4" />
                                Archive
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </CardHeader>

            <CardContent className="pb-3">
                {/* Stats */}
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <div className="flex items-center gap-1">
                        <FolderOpen className="h-4 w-4" />
                        <span>0 envs</span> {/* TODO: Add real environment count */}
                    </div>
                    <div className="flex items-center gap-1">
                        <span>0 flags</span> {/* TODO: Add real flag count */}
                    </div>
                </div>
            </CardContent>

            <CardFooter className="pt-0">
                <div className="flex items-center justify-between w-full">
                    <Badge variant="secondary" className="text-xs">
                        Updated {formatDate(project.updated_at)}
                    </Badge>

                    <Button
                        variant="ghost"
                        size="sm"
                        className="opacity-0 group-hover:opacity-100 transition-opacity gap-2"
                        onClick={(e) => {
                            e.stopPropagation();
                            onOpen(project);
                        }}
                    >
                        <ExternalLink className="h-4 w-4" />
                        Open
                    </Button>
                </div>
            </CardFooter>
        </Card>
    );
}
