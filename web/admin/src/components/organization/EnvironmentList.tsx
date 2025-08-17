'use client';

import { CopyButton } from '@/components/primitives';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Switch } from '@/components/ui/switch';
import { Edit, MoreHorizontal, Settings, Trash2 } from 'lucide-react';
import { memo, useState } from 'react';
import { EmptyState } from './EmptyState';

interface Environment {
    id: string;
    name: string;
    key: string;
    description?: string;
    created_at: string;
    enabled?: boolean;
    isDefault?: boolean;
}

interface EnvironmentListProps {
    environments: Environment[];
    selectedEnvKey?: string | null;
    onSelectEnvironment: (environment: Environment) => void;
    onToggleEnvironment?: (environment: Environment) => Promise<void>;
    onEditEnvironment?: (environment: Environment) => void;
    onDeleteEnvironment?: (environment: Environment) => Promise<void>;
}

const EnvironmentItem = memo(({
    environment,
    isSelected,
    onSelect,
    onToggle,
    onEdit,
    onDelete
}: {
    environment: Environment;
    isSelected: boolean;
    onSelect: () => void;
    onToggle?: () => Promise<void>;
    onEdit?: () => void;
    onDelete?: () => Promise<void>;
}) => {
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const [isDeleting, setIsDeleting] = useState(false);

    const handleDelete = async () => {
        if (!onDelete) return;
        setIsDeleting(true);
        try {
            await onDelete();
            setShowDeleteDialog(false);
        } finally {
            setIsDeleting(false);
        }
    };

    return (
        <>
            <div
                className={`group relative p-3 border rounded-lg cursor-pointer transition-all duration-200 ${isSelected
                    ? 'bg-muted/50 border-muted-foreground/20 shadow-sm ring-1 ring-muted-foreground/20'
                    : 'hover:bg-muted/30 hover:border-muted-foreground/10'
                    }`}
                onClick={onSelect}
            >
                <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                            <h3 className="font-medium text-sm line-clamp-1">{environment.name}</h3>
                            {environment.isDefault && (
                                <Badge variant="secondary" className="text-xs">Default</Badge>
                            )}
                        </div>

                        <div className="flex items-center gap-2 mb-2">
                            <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                                {environment.key}
                            </code>
                            <CopyButton text={environment.key} />
                        </div>

                        {onToggle && (
                            <div className="flex items-center gap-2 mb-2">
                                <Switch
                                    checked={environment.enabled ?? true}
                                    onCheckedChange={onToggle}
                                    onClick={(e) => e.stopPropagation()}
                                />
                                <span className="text-xs text-muted-foreground">
                                    {environment.enabled ?? true ? 'Enabled' : 'Disabled'}
                                </span>
                            </div>
                        )}
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
                            {onEdit && (
                                <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onEdit(); }}>
                                    <Edit className="h-4 w-4 mr-2" />
                                    Rename
                                </DropdownMenuItem>
                            )}
                            {onDelete && !environment.isDefault && (
                                <DropdownMenuItem
                                    onClick={(e) => { e.stopPropagation(); setShowDeleteDialog(true); }}
                                    className="text-destructive"
                                >
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    Delete
                                </DropdownMenuItem>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>

            <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Environment</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete "{environment.name}"? This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={handleDelete}
                            disabled={isDeleting}
                        >
                            {isDeleting ? 'Deleting...' : 'Delete'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    );
});

EnvironmentItem.displayName = 'EnvironmentItem';

export function EnvironmentList({
    environments,
    selectedEnvKey,
    onSelectEnvironment,
    onToggleEnvironment,
    onEditEnvironment,
    onDeleteEnvironment
}: EnvironmentListProps) {
    if (environments.length === 0) {
        return (
            <EmptyState
                icon={Settings}
                title="No environments yet"
                description="Create your first environment above."
            />
        );
    }

    return (
        <div className="space-y-2">
            {environments.map((environment) => (
                <EnvironmentItem
                    key={environment.id}
                    environment={environment}
                    isSelected={selectedEnvKey === environment.key}
                    onSelect={() => onSelectEnvironment(environment)}
                    onToggle={onToggleEnvironment ? () => onToggleEnvironment(environment) : undefined}
                    onEdit={onEditEnvironment ? () => onEditEnvironment(environment) : undefined}
                    onDelete={onDeleteEnvironment ? () => onDeleteEnvironment(environment) : undefined}
                />
            ))}
        </div>
    );
}
