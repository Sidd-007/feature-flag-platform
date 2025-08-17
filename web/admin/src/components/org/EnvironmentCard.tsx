'use client';

import { CopyButton } from '@/components/primitives';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { MoreHorizontal, ExternalLink, Edit, Trash2, Flag } from 'lucide-react';
import { useState } from 'react';

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

interface EnvironmentCardProps {
    environment: Environment;
    projectId: string;
    onOpenFlags: (env: Environment) => void;
    onToggleEnabled: (env: Environment) => void;
    onEdit: (env: Environment) => void;
    onDelete: (env: Environment) => void;
}

export function EnvironmentCard({ 
    environment, 
    projectId, 
    onOpenFlags, 
    onToggleEnabled, 
    onEdit, 
    onDelete 
}: EnvironmentCardProps) {
    const [isContextMenuOpen, setIsContextMenuOpen] = useState(false);
    const [isToggling, setIsToggling] = useState(false);

    const handleContextMenu = (e: React.MouseEvent) => {
        e.preventDefault();
        setIsContextMenuOpen(true);
    };

    const handleToggle = async () => {
        setIsToggling(true);
        try {
            await onToggleEnabled(environment);
        } finally {
            setIsToggling(false);
        }
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
            onClick={() => onOpenFlags(environment)}
            onContextMenu={handleContextMenu}
        >
            <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                            <h3 className="font-semibold text-lg truncate">
                                {environment.name}
                            </h3>
                            
                            {/* Status Indicator */}
                            <div className={`w-2 h-2 rounded-full ${
                                environment.enabled !== false ? 'bg-green-500' : 'bg-gray-400'
                            }`} />
                            
                            {environment.isDefault && (
                                <Badge variant="secondary" className="text-xs">
                                    Default
                                </Badge>
                            )}
                        </div>

                        {environment.description && (
                            <p className="text-sm text-muted-foreground line-clamp-2 mb-3">
                                {environment.description}
                            </p>
                        )}
                        
                        {/* Environment Key with Copy Button */}
                        <div className="flex items-center gap-2">
                            <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                                {environment.key}
                            </code>
                            <CopyButton text={environment.key} />
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
                                    onOpenFlags(environment);
                                }}
                                className="gap-2"
                            >
                                <Flag className="h-4 w-4" />
                                Open Flags
                            </DropdownMenuItem>
                            <DropdownMenuItem 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onEdit(environment);
                                }}
                                className="gap-2"
                            >
                                <Edit className="h-4 w-4" />
                                Edit
                            </DropdownMenuItem>
                            <DropdownMenuItem 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onDelete(environment);
                                }}
                                className="gap-2 text-destructive focus:text-destructive"
                            >
                                <Trash2 className="h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </CardHeader>

            <CardContent className="pb-3">
                {/* Enabled Toggle */}
                <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Enabled</span>
                    <div className="flex items-center gap-2">
                        <Switch
                            checked={environment.enabled !== false}
                            onCheckedChange={handleToggle}
                            disabled={isToggling}
                            onClick={(e) => e.stopPropagation()}
                        />
                        <span className="text-sm font-medium">
                            {environment.enabled !== false ? 'ON' : 'OFF'}
                        </span>
                    </div>
                </div>
            </CardContent>

            <CardFooter className="pt-0">
                <div className="flex items-center justify-between w-full">
                    <Badge variant="secondary" className="text-xs">
                        Updated {formatDate(environment.updated_at)}
                    </Badge>
                    
                    <Button
                        variant="ghost"
                        size="sm"
                        className="opacity-0 group-hover:opacity-100 transition-opacity gap-2"
                        onClick={(e) => {
                            e.stopPropagation();
                            onOpenFlags(environment);
                        }}
                    >
                        <Flag className="h-4 w-4" />
                        Open Flags
                    </Button>
                </div>
            </CardFooter>
        </Card>
    );
}
