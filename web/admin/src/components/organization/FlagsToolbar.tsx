'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Plus, Search } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

interface Environment {
    id: string;
    name: string;
    key: string;
}

interface FlagsToolbarProps {
    searchQuery: string;
    onSearchChange: (query: string) => void;
    typeFilter: string;
    onTypeChange: (type: string) => void;
    statusFilter: string;
    onStatusChange: (status: string) => void;
    selectedEnvKey?: string | null;
    environments: Environment[];
    onEnvChange: (envKey: string) => void;
    onCreateFlag: () => void;
    canCreateFlag: boolean;
}

export function FlagsToolbar({
    searchQuery,
    onSearchChange,
    typeFilter,
    onTypeChange,
    statusFilter,
    onStatusChange,
    selectedEnvKey,
    environments,
    onEnvChange,
    onCreateFlag,
    canCreateFlag
}: FlagsToolbarProps) {
    const [localSearch, setLocalSearch] = useState(searchQuery);
    const searchInputRef = useRef<HTMLInputElement>(null);
    const debounceRef = useRef<NodeJS.Timeout>();

    // Debounced search
    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            onSearchChange(localSearch);
        }, 250);

        return () => clearTimeout(debounceRef.current);
    }, [localSearch, onSearchChange]);

    // Keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // "/" to focus search
            if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
                e.preventDefault();
                searchInputRef.current?.focus();
            }

            // "n" to open create flag dialog (when not typing)
            if (e.key === 'n' && !e.ctrlKey && !e.metaKey && !e.altKey &&
                e.target instanceof HTMLElement &&
                !['INPUT', 'TEXTAREA'].includes(e.target.tagName) &&
                canCreateFlag) {
                e.preventDefault();
                onCreateFlag();
            }

            // Escape to clear search
            if (e.key === 'Escape' && searchInputRef.current === document.activeElement) {
                setLocalSearch('');
                searchInputRef.current?.blur();
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onCreateFlag, canCreateFlag]);

    return (
        <div className="sticky top-0 bg-card border-b border-border p-4 -m-4 mb-4 z-10">
            <div className="flex flex-col md:flex-row gap-4 items-start md:items-center justify-between">
                <div className="flex flex-1 gap-3 w-full md:w-auto">
                    <div className="relative flex-1 md:w-64">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                            ref={searchInputRef}
                            placeholder="Search flags... (Press / to focus)"
                            value={localSearch}
                            onChange={(e) => setLocalSearch(e.target.value)}
                            className="h-9 pl-10"
                        />
                    </div>

                    <Select value={typeFilter} onValueChange={onTypeChange}>
                        <SelectTrigger className="h-9 w-32">
                            <SelectValue placeholder="Type" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All Types</SelectItem>
                            <SelectItem value="boolean">Boolean</SelectItem>
                            <SelectItem value="string">String</SelectItem>
                            <SelectItem value="number">Number</SelectItem>
                            <SelectItem value="json">JSON</SelectItem>
                            <SelectItem value="multivariate">Multivariate</SelectItem>
                        </SelectContent>
                    </Select>

                    <Select value={statusFilter} onValueChange={onStatusChange}>
                        <SelectTrigger className="h-9 w-32">
                            <SelectValue placeholder="Status" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All Status</SelectItem>
                            <SelectItem value="published">Published</SelectItem>
                            <SelectItem value="draft">Draft</SelectItem>
                        </SelectContent>
                    </Select>

                    {environments.length > 0 && (
                        <Select value={selectedEnvKey || ''} onValueChange={onEnvChange}>
                            <SelectTrigger className="h-9 w-40">
                                <SelectValue placeholder="Environment" />
                            </SelectTrigger>
                            <SelectContent>
                                {environments.map((env) => (
                                    <SelectItem key={env.id} value={env.key}>
                                        {env.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    )}
                </div>

                {canCreateFlag && (
                    <Button onClick={onCreateFlag} className="gap-2 whitespace-nowrap">
                        <Plus className="h-4 w-4" />
                        Create Flag
                        <kbd className="hidden md:inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
                            N
                        </kbd>
                    </Button>
                )}
            </div>
        </div>
    );
}
