"use client";

import { CopyButton, EmptyState } from '@/components/primitives';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { useToast } from '@/hooks/use-toast';
import apiClient from '@/lib/api';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, Eye, EyeOff, KeyRound, Plus, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

interface APIToken {
    id: string;
    name: string;
    description?: string;
    scope: string;
    prefix: string;
    expires_at?: string;
    created_at: string;
    last_used_at?: string;
    is_active: boolean;
}

interface CreateTokenResponse {
    token: APIToken;
    plain_key: string;
}

interface ApiTokensSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    orgId: string;
    projectId?: string;
    envId?: string;
}

const tokenSchema = z.object({
    name: z.string().min(1, 'Token name is required'),
    description: z.string().optional(),
    scope: z.enum(['read', 'write']),
});

type TokenData = z.infer<typeof tokenSchema>;

export function ApiTokensSheet({ open, onOpenChange, orgId, projectId, envId }: ApiTokensSheetProps) {
    const { toast } = useToast();
    const [tokens, setTokens] = useState<APIToken[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreateDialog, setShowCreateDialog] = useState(false);
    const [newToken, setNewToken] = useState<CreateTokenResponse | null>(null);
    const [showNewTokenDialog, setShowNewTokenDialog] = useState(false);
    const [revealedTokens, setRevealedTokens] = useState<Set<string>>(new Set());

    const form = useForm<TokenData>({
        resolver: zodResolver(tokenSchema),
        defaultValues: {
            name: '',
            description: '',
            scope: 'read',
        },
    });

    const loadTokens = async () => {
        if (!projectId || !envId) return;

        setLoading(true);
        try {
            const response = await apiClient.get(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens`);
            if (response.error) {
                toast({
                    title: "Error loading tokens",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                let tokensData = response.data;
                if (tokensData && typeof tokensData === 'object' && !Array.isArray(tokensData)) {
                    const dataObj = tokensData as any;
                    tokensData = dataObj.data || dataObj.tokens || dataObj.items || tokensData;
                }
                const tokensArray = Array.isArray(tokensData) ? tokensData : [];
                setTokens(tokensArray);
            }
        } catch (err) {
            toast({
                title: "Failed to load tokens",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        } finally {
            setLoading(false);
        }
    };

    const createToken = async (data: TokenData) => {
        if (!projectId || !envId) return;

        try {
            const response = await apiClient.post(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens`, data);

            if (response.error) {
                toast({
                    title: "Failed to create token",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                setNewToken(response.data as CreateTokenResponse);
                setShowNewTokenDialog(true);
                setShowCreateDialog(false);
                form.reset();
                await loadTokens();
            }
        } catch (err) {
            toast({
                title: "Failed to create token",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const revokeToken = async (tokenId: string) => {
        if (!projectId || !envId) return;

        try {
            const response = await apiClient.delete(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens/${tokenId}`);

            if (response.error) {
                toast({
                    title: "Failed to revoke token",
                    description: response.error,
                    variant: "destructive",
                });
            } else {
                toast({
                    title: "Success",
                    description: "Token revoked successfully",
                });
                await loadTokens();
            }
        } catch (err) {
            toast({
                title: "Failed to revoke token",
                description: "An unexpected error occurred",
                variant: "destructive",
            });
        }
    };

    const toggleTokenVisibility = (tokenId: string) => {
        setRevealedTokens(prev => {
            const newSet = new Set(prev);
            if (newSet.has(tokenId)) {
                newSet.delete(tokenId);
            } else {
                newSet.add(tokenId);
            }
            return newSet;
        });
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    useEffect(() => {
        if (open && projectId && envId) {
            loadTokens();
        }
    }, [open, orgId, projectId, envId]);

    return (
        <>
            <Sheet open={open} onOpenChange={onOpenChange}>
                <SheetContent className="w-[700px] sm:w-[600px] max-w-[90vw]">
                    <SheetHeader>
                        <SheetTitle className="flex items-center gap-2">
                            <KeyRound className="h-5 w-5" />
                            API Tokens
                        </SheetTitle>
                        <SheetDescription>
                            Manage API tokens for this environment. Tokens are used to authenticate with the Feature Flags platform.
                        </SheetDescription>
                    </SheetHeader>

                    <div className="mt-6 space-y-6">
                        {!projectId || !envId ? (
                            <Alert>
                                <AlertCircle className="h-4 w-4" />
                                <AlertDescription>
                                    Select a project and environment to manage API tokens.
                                </AlertDescription>
                            </Alert>
                        ) : (
                            <>
                                <div className="flex justify-between items-center">
                                    <div>
                                        <h3 className="text-lg font-semibold">Active Tokens</h3>
                                        <p className="text-sm text-muted-foreground">
                                            {tokens.filter(t => t.is_active).length} active token{tokens.filter(t => t.is_active).length !== 1 ? 's' : ''}
                                        </p>
                                    </div>
                                    <Button onClick={() => setShowCreateDialog(true)} className="gap-2">
                                        <Plus className="h-4 w-4" />
                                        Create Token
                                    </Button>
                                </div>

                                {loading ? (
                                    <div className="space-y-3">
                                        {[...Array(3)].map((_, i) => (
                                            <Skeleton key={i} className="h-20 w-full" />
                                        ))}
                                    </div>
                                ) : tokens.length === 0 ? (
                                    <EmptyState
                                        icon={KeyRound}
                                        title="No API tokens"
                                        description="Create your first API token to start using the Feature Flags platform."
                                        action={{
                                            label: "Create Token",
                                            onClick: () => setShowCreateDialog(true)
                                        }}
                                    />
                                ) : (
                                    <div className="space-y-4">
                                        {tokens.map((token) => (
                                            <Card key={token.id}>
                                                <CardContent className="pt-4">
                                                    <div className="flex items-start justify-between">
                                                        <div className="space-y-2 flex-1">
                                                            <div className="flex items-center gap-2">
                                                                <h4 className="font-medium">{token.name}</h4>
                                                                <Badge variant={token.scope === 'write' ? 'default' : 'secondary'}>
                                                                    {token.scope}
                                                                </Badge>
                                                                {!token.is_active && (
                                                                    <Badge variant="destructive">Revoked</Badge>
                                                                )}
                                                            </div>
                                                            {token.description && (
                                                                <p className="text-sm text-muted-foreground">
                                                                    {token.description}
                                                                </p>
                                                            )}
                                                            <div className="flex items-center gap-2">
                                                                <code className="text-xs bg-muted px-2 py-1 rounded">
                                                                    {revealedTokens.has(token.id)
                                                                        ? `${token.prefix}${'*'.repeat(32)}`
                                                                        : `${token.prefix}***`
                                                                    }
                                                                </code>
                                                                <Button
                                                                    variant="ghost"
                                                                    size="sm"
                                                                    onClick={() => toggleTokenVisibility(token.id)}
                                                                >
                                                                    {revealedTokens.has(token.id) ? (
                                                                        <EyeOff className="h-3 w-3" />
                                                                    ) : (
                                                                        <Eye className="h-3 w-3" />
                                                                    )}
                                                                </Button>
                                                                <CopyButton text={token.prefix} />
                                                            </div>
                                                            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                                                                <span>Created: {formatDate(token.created_at)}</span>
                                                                {token.last_used_at && (
                                                                    <span>Last used: {formatDate(token.last_used_at)}</span>
                                                                )}
                                                            </div>
                                                        </div>

                                                        {token.is_active && (
                                                            <Button
                                                                variant="destructive"
                                                                size="sm"
                                                                onClick={() => revokeToken(token.id)}
                                                                className="gap-1"
                                                            >
                                                                <Trash2 className="h-3 w-3" />
                                                                Revoke
                                                            </Button>
                                                        )}
                                                    </div>
                                                </CardContent>
                                            </Card>
                                        ))}
                                    </div>
                                )}

                                <Separator />

                                {/* Usage Instructions */}
                                <div className="space-y-4">
                                    <h3 className="text-lg font-semibold">Usage Instructions</h3>
                                    <div className="space-y-4">
                                        <div>
                                            <h4 className="font-medium mb-2">Go SDK</h4>
                                            <pre className="bg-muted p-3 rounded text-sm overflow-x-auto text-muted-foreground">
                                                {`config := &featureflags.Config{
    APIKey:      "your_api_token_here",
    Environment: "${envId || 'your_env_key'}",
    EvaluatorEndpoint: "http://localhost:8081",
}`}
                                            </pre>
                                        </div>

                                        <div>
                                            <h4 className="font-medium mb-2">HTTP API</h4>
                                            <pre className="bg-muted p-3 rounded text-sm overflow-x-auto text-muted-foreground">
                                                {`curl -X POST http://localhost:8081/v1/evaluate \\
  -H "Authorization: Bearer your_api_token_here" \\
  -H "Content-Type: application/json" \\
  -d '{
    "env_key": "${envId || 'your_env_key'}",
    "context": {
      "user_key": "user-123",
      "attributes": {"plan": "premium"}
    }
  }'`}
                                            </pre>
                                        </div>

                                        <Alert>
                                            <AlertCircle className="h-4 w-4" />
                                            <AlertDescription className="text-sm">
                                                <strong>Read tokens</strong> can only evaluate flags and retrieve configurations.
                                                <strong> Write tokens</strong> can also modify flags and configurations (future feature).
                                            </AlertDescription>
                                        </Alert>
                                    </div>
                                </div>
                            </>
                        )}
                    </div>
                </SheetContent>
            </Sheet>

            {/* Create Token Dialog */}
            <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>Create API Token</DialogTitle>
                        <DialogDescription>
                            Create a new API token for this environment
                        </DialogDescription>
                    </DialogHeader>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(createToken)} className="space-y-4">
                            <FormField
                                control={form.control}
                                name="name"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Token Name</FormLabel>
                                        <FormControl>
                                            <Input placeholder="Production SDK Token" {...field} className="h-9" />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="description"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Description (Optional)</FormLabel>
                                        <FormControl>
                                            <Textarea
                                                placeholder="Describe what this token is used for..."
                                                {...field}
                                                className="min-h-[60px]"
                                            />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="scope"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Scope</FormLabel>
                                        <FormControl>
                                            <Select value={field.value} onValueChange={field.onChange}>
                                                <SelectTrigger className="h-9">
                                                    <SelectValue />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="read">Read Only</SelectItem>
                                                    <SelectItem value="write">Read & Write</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <DialogFooter>
                                <Button type="button" variant="outline" onClick={() => setShowCreateDialog(false)}>
                                    Cancel
                                </Button>
                                <Button type="submit" disabled={form.formState.isSubmitting}>
                                    {form.formState.isSubmitting ? 'Creating...' : 'Create Token'}
                                </Button>
                            </DialogFooter>
                        </form>
                    </Form>
                </DialogContent>
            </Dialog>

            {/* New Token Display Dialog */}
            {newToken && (
                <Dialog open={showNewTokenDialog} onOpenChange={setShowNewTokenDialog}>
                    <DialogContent className="sm:max-w-lg">
                        <DialogHeader>
                            <DialogTitle className="text-green-800">Token Created Successfully!</DialogTitle>
                            <DialogDescription className="text-green-700">
                                Please copy this token now. You won't be able to see it again.
                            </DialogDescription>
                        </DialogHeader>
                        <div className="space-y-4">
                            <div className="flex items-center gap-2">
                                <Input
                                    value={newToken.plain_key}
                                    readOnly
                                    className="font-mono text-sm bg-green-50 border-green-200"
                                />
                                <CopyButton text={newToken.plain_key} />
                            </div>
                            <Alert className="border-green-200 bg-green-50">
                                <AlertCircle className="h-4 w-4 text-green-600" />
                                <AlertDescription className="text-green-700">
                                    Make sure to save this token in a secure location. It cannot be recovered once this dialog is closed.
                                </AlertDescription>
                            </Alert>
                        </div>
                        <DialogFooter>
                            <Button
                                onClick={() => {
                                    setShowNewTokenDialog(false);
                                    setNewToken(null);
                                }}
                                className="w-full"
                            >
                                I've copied the token
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            )}
        </>
    );
}
