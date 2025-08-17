'use client';

import { CopyButton } from '@/components/primitives';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch } from '@/components/ui/switch';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { CheckCircle2, ChevronLeft, ChevronRight, Copy, Edit, Loader2, MoreHorizontal, Power, Trash2, Zap } from 'lucide-react';
import { memo, useMemo, useState } from 'react';

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

interface FlagsDataTableProps {
    flags: Flag[];
    isLoading: boolean;
    currentPage: number;
    pageSize: number;
    totalFlags: number;
    onPageChange: (page: number) => void;
    onPageSizeChange: (size: number) => void;
    onToggleFlag: (flag: Flag) => Promise<void>;
    onPublishFlag: (flag: Flag) => Promise<void>;
    onEditFlag?: (flag: Flag) => void;
    onDuplicateFlag?: (flag: Flag) => void;
    onDeleteFlag?: (flag: Flag) => void;
}

const FlagTableRow = memo(({
    flag,
    onToggle,
    onPublish,
    onEdit,
    onDuplicate,
    onDelete,
    style
}: {
    flag: Flag;
    onToggle: () => Promise<void>;
    onPublish: () => Promise<void>;
    onEdit?: () => void;
    onDuplicate?: () => void;
    onDelete?: () => void;
    style?: React.CSSProperties;
}) => {
    const [isToggling, setIsToggling] = useState(false);

    const handleToggle = async () => {
        setIsToggling(true);
        try {
            await onToggle();
        } finally {
            setIsToggling(false);
        }
    };

    return (
        <TableRow
            className="hover:bg-muted/50 h-12"
            style={style}
        >
            <TableCell className="max-w-0">
                <div className="min-w-0">
                    <div className="font-medium text-sm line-clamp-1">{flag.name}</div>
                    {flag.description && (
                        <div className="text-xs text-muted-foreground line-clamp-1 mt-1">
                            {flag.description}
                        </div>
                    )}
                </div>
            </TableCell>

            <TableCell>
                <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                        {flag.key}
                    </code>
                    <CopyButton text={flag.key} />
                </div>
            </TableCell>

            <TableCell>
                <Badge variant="secondary" className="text-xs">
                    {flag.type}
                </Badge>
            </TableCell>

            <TableCell>
                {flag.published ? (
                    <Badge variant="outline" className="text-xs gap-1">
                        <CheckCircle2 className="h-3 w-3" />
                        Published v{flag.version}
                    </Badge>
                ) : (
                    <Badge variant="secondary" className="text-xs">
                        Draft
                    </Badge>
                )}
            </TableCell>

            <TableCell>
                <div className="flex items-center gap-2">
                    <Switch
                        checked={flag.enabled}
                        onCheckedChange={handleToggle}
                        disabled={isToggling}
                    />
                    <span className="text-sm">
                        {flag.enabled ? 'ON' : 'OFF'}
                    </span>
                </div>
            </TableCell>

            <TableCell>
                <div className="text-sm text-muted-foreground">
                    {new Date(flag.created_at).toLocaleDateString()}
                </div>
            </TableCell>

            <TableCell className="text-right">
                <div className="flex items-center justify-end gap-2">
                    <Button
                        size="sm"
                        variant={flag.published ? "destructive" : "outline"}
                        onClick={onPublish}
                        disabled={flag.isPublishing}
                        className="gap-1"
                    >
                        {flag.isPublishing ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                        ) : flag.published ? (
                            <>
                                <Power className="h-3 w-3" />
                                Unpublish
                            </>
                        ) : (
                            <>
                                <Zap className="h-3 w-3" />
                                Publish
                            </>
                        )}
                    </Button>

                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            {onEdit && (
                                <DropdownMenuItem onClick={onEdit}>
                                    <Edit className="h-4 w-4 mr-2" />
                                    Edit
                                </DropdownMenuItem>
                            )}
                            {onDuplicate && (
                                <DropdownMenuItem onClick={onDuplicate}>
                                    <Copy className="h-4 w-4 mr-2" />
                                    Duplicate
                                </DropdownMenuItem>
                            )}
                            {onDelete && (
                                <DropdownMenuItem onClick={onDelete} className="text-destructive">
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    Delete
                                </DropdownMenuItem>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </TableCell>
        </TableRow>
    );
});

FlagTableRow.displayName = 'FlagTableRow';

const LoadingSkeleton = memo(() => (
    <TableRow className="h-12">
        {Array.from({ length: 7 }).map((_, i) => (
            <TableCell key={i}>
                <Skeleton className="h-4 w-full" />
            </TableCell>
        ))}
    </TableRow>
));

LoadingSkeleton.displayName = 'LoadingSkeleton';

export function FlagsDataTable({
    flags,
    isLoading,
    currentPage,
    pageSize,
    totalFlags,
    onPageChange,
    onPageSizeChange,
    onToggleFlag,
    onPublishFlag,
    onEditFlag,
    onDuplicateFlag,
    onDeleteFlag
}: FlagsDataTableProps) {
    // Performance optimization: Use CSS-based scrolling with sticky header for large lists
    // This provides good performance without external dependencies
    // For 50+ items, we use a scrollable container with fixed height
    const shouldUseScrollContainer = flags.length > 50;

    const totalPages = Math.ceil(totalFlags / pageSize);
    const startIndex = (currentPage - 1) * pageSize + 1;
    const endIndex = Math.min(currentPage * pageSize, totalFlags);

    const paginationInfo = useMemo(() => ({
        start: startIndex,
        end: endIndex,
        total: totalFlags,
        totalPages
    }), [startIndex, endIndex, totalFlags, totalPages]);

    if (isLoading) {
        return (
            <div className="rounded-md border">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Key</TableHead>
                            <TableHead>Type</TableHead>
                            <TableHead>Version</TableHead>
                            <TableHead>State</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {Array.from({ length: pageSize }).map((_, i) => (
                            <LoadingSkeleton key={i} />
                        ))}
                    </TableBody>
                </Table>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="rounded-md border">
                <div className={shouldUseScrollContainer ? "max-h-[600px] overflow-auto" : ""}>
                    <Table>
                        <TableHeader className={shouldUseScrollContainer ? "sticky top-0 bg-background z-10" : ""}>
                            <TableRow>
                                <TableHead>Name</TableHead>
                                <TableHead>Key</TableHead>
                                <TableHead>Type</TableHead>
                                <TableHead>Version</TableHead>
                                <TableHead>State</TableHead>
                                <TableHead>Updated</TableHead>
                                <TableHead className="text-right">Actions</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {flags.map((flag) => (
                                <FlagTableRow
                                    key={flag.id}
                                    flag={flag}
                                    onToggle={() => onToggleFlag(flag)}
                                    onPublish={() => onPublishFlag(flag)}
                                    onEdit={onEditFlag ? () => onEditFlag(flag) : undefined}
                                    onDuplicate={onDuplicateFlag ? () => onDuplicateFlag(flag) : undefined}
                                    onDelete={onDeleteFlag ? () => onDeleteFlag(flag) : undefined}
                                />
                            ))}
                        </TableBody>
                    </Table>
                </div>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>
                        Showing {paginationInfo.start} to {paginationInfo.end} of {paginationInfo.total} flags
                    </span>
                    <Select value={pageSize.toString()} onValueChange={(value) => onPageSizeChange(Number(value))}>
                        <SelectTrigger className="h-8 w-20">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="25">25</SelectItem>
                            <SelectItem value="50">50</SelectItem>
                            <SelectItem value="100">100</SelectItem>
                        </SelectContent>
                    </Select>
                    <span>per page</span>
                </div>

                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onPageChange(currentPage - 1)}
                        disabled={currentPage <= 1}
                        className="gap-1"
                    >
                        <ChevronLeft className="h-4 w-4" />
                        Previous
                    </Button>

                    <div className="flex items-center gap-1">
                        {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                            let pageNum;
                            if (totalPages <= 5) {
                                pageNum = i + 1;
                            } else if (currentPage <= 3) {
                                pageNum = i + 1;
                            } else if (currentPage >= totalPages - 2) {
                                pageNum = totalPages - 4 + i;
                            } else {
                                pageNum = currentPage - 2 + i;
                            }

                            return (
                                <Button
                                    key={pageNum}
                                    variant={pageNum === currentPage ? "default" : "outline"}
                                    size="sm"
                                    onClick={() => onPageChange(pageNum)}
                                    className="w-8 h-8 p-0"
                                >
                                    {pageNum}
                                </Button>
                            );
                        })}
                    </div>

                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onPageChange(currentPage + 1)}
                        disabled={currentPage >= totalPages}
                        className="gap-1"
                    >
                        Next
                        <ChevronRight className="h-4 w-4" />
                    </Button>
                </div>
            </div>
        </div>
    );
}
