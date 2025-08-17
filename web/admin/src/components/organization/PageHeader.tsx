'use client';

import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/components/ui/breadcrumb';
import { Button } from '@/components/ui/button';
import { ArrowLeft, KeyRound } from 'lucide-react';

interface PageHeaderProps {
    orgName?: string;
    onOpenTokensSheet: () => void;
    onNavigateBack: () => void;
}

export function PageHeader({ orgName, onOpenTokensSheet, onNavigateBack }: PageHeaderProps) {
    return (
        <div className="mx-auto max-w-[1200px] px-6 pt-8 pb-6">
            {/* Breadcrumbs */}
            <div className="mb-6">
                <Breadcrumb>
                    <BreadcrumbList>
                        <BreadcrumbItem>
                            <BreadcrumbLink href="/">Organizations</BreadcrumbLink>
                        </BreadcrumbItem>
                        <BreadcrumbSeparator />
                        <BreadcrumbItem>
                            <BreadcrumbPage>{orgName || 'Management'}</BreadcrumbPage>
                        </BreadcrumbItem>
                        <BreadcrumbSeparator />
                        <BreadcrumbItem>
                            <BreadcrumbPage>Management</BreadcrumbPage>
                        </BreadcrumbItem>
                    </BreadcrumbList>
                </Breadcrumb>
            </div>

            {/* Header with actions */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Organization Management</h1>
                    <p className="text-muted-foreground mt-2">
                        Manage your projects, environments, and feature flags
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <Button
                        variant="outline"
                        onClick={onOpenTokensSheet}
                        className="gap-2"
                    >
                        <KeyRound className="h-4 w-4" />
                        Manage API Tokens
                    </Button>
                    <Button
                        variant="outline"
                        onClick={onNavigateBack}
                        className="gap-2"
                    >
                        <ArrowLeft className="h-4 w-4" />
                        Back to Organizations
                    </Button>
                </div>
            </div>
        </div>
    );
}
