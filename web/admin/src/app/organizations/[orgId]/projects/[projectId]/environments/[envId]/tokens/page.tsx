"use client";

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { toast } from '@/components/ui/use-toast';
import { apiClient as api } from '@/lib/api';
import { Copy, Eye, EyeOff, Plus, Trash2 } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

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

export default function APITokensPage() {
    const params = useParams();
    const router = useRouter();
    const { orgId, projectId, envId } = params;

    const [tokens, setTokens] = useState<APIToken[]>([]);
    const [loading, setLoading] = useState(true);
    const [creating, setCreating] = useState(false);
    const [showDialog, setShowDialog] = useState(false);
    const [showKey, setShowKey] = useState<string | null>(null);
    const [newToken, setNewToken] = useState<CreateTokenResponse | null>(null);

    const [form, setForm] = useState({
        name: '',
        description: '',
        scope: 'read'
    });

    const loadTokens = async () => {
        try {
            const response = await api.get(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens`);
            setTokens((response.data as any)?.data || []);
        } catch (error: any) {
            console.error('Failed to load API tokens:', error);
            toast({
                title: "Error",
                description: "Failed to load API tokens",
                variant: "destructive",
            });
        } finally {
            setLoading(false);
        }
    };

    const createToken = async () => {
        setCreating(true);
        try {
            const response = await api.post(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens`, form);
            setNewToken(response.data as CreateTokenResponse);
            setForm({ name: '', description: '', scope: 'read' });
            setShowDialog(false);
            await loadTokens();

            toast({
                title: "Success",
                description: "API token created successfully",
            });
        } catch (error: any) {
            console.error('Failed to create API token:', error);
            toast({
                title: "Error",
                description: (error as any).response?.data?.error || "Failed to create API token",
                variant: "destructive",
            });
        } finally {
            setCreating(false);
        }
    };

    const revokeToken = async (tokenId: string) => {
        try {
            await api.delete(`/orgs/${orgId}/projects/${projectId}/environments/${envId}/tokens/${tokenId}`);
            await loadTokens();

            toast({
                title: "Success",
                description: "API token revoked successfully",
            });
        } catch (error: any) {
            console.error('Failed to revoke API token:', error);
            toast({
                title: "Error",
                description: "Failed to revoke API token",
                variant: "destructive",
            });
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        toast({
            title: "Copied",
            description: "API token copied to clipboard",
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
        loadTokens();
    }, [orgId, projectId, envId]);

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-muted-foreground">Loading API tokens...</div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold">API Tokens</h1>
                    <p className="text-muted-foreground">
                        Manage API tokens for this environment
                    </p>
                </div>

                <Dialog open={showDialog} onOpenChange={setShowDialog}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="h-4 w-4 mr-2" />
                            Create Token
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Create API Token</DialogTitle>
                            <DialogDescription>
                                Create a new API token for this environment
                            </DialogDescription>
                        </DialogHeader>

                        <div className="space-y-4">
                            <div>
                                <Label htmlFor="name">Name</Label>
                                <Input
                                    id="name"
                                    placeholder="e.g., Production SDK Token"
                                    value={form.name}
                                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                                />
                            </div>

                            <div>
                                <Label htmlFor="description">Description (optional)</Label>
                                <Textarea
                                    id="description"
                                    placeholder="Describe what this token is used for..."
                                    value={form.description}
                                    onChange={(e) => setForm({ ...form, description: e.target.value })}
                                />
                            </div>

                            <div>
                                <Label htmlFor="scope">Scope</Label>
                                <Select value={form.scope} onValueChange={(value) => setForm({ ...form, scope: value })}>
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="read">Read Only</SelectItem>
                                        <SelectItem value="write">Read & Write</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="flex justify-end space-x-2">
                                <Button variant="outline" onClick={() => setShowDialog(false)}>
                                    Cancel
                                </Button>
                                <Button onClick={createToken} disabled={creating || !form.name}>
                                    {creating ? 'Creating...' : 'Create Token'}
                                </Button>
                            </div>
                        </div>
                    </DialogContent>
                </Dialog>
            </div>

            {/* New Token Display */}
            {newToken && (
                <Card className="border-green-200 bg-green-50">
                    <CardHeader>
                        <CardTitle className="text-green-800">Token Created Successfully!</CardTitle>
                        <CardDescription className="text-green-700">
                            Please copy this token now. You won't be able to see it again.
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="flex items-center space-x-2">
                            <Input
                                value={newToken.plain_key}
                                readOnly
                                className="font-mono text-sm"
                            />
                            <Button
                                size="sm"
                                onClick={() => copyToClipboard(newToken.plain_key)}
                            >
                                <Copy className="h-4 w-4" />
                            </Button>
                        </div>
                        <Button
                            variant="outline"
                            size="sm"
                            className="mt-2"
                            onClick={() => setNewToken(null)}
                        >
                            I've copied the token
                        </Button>
                    </CardContent>
                </Card>
            )}

            {/* Tokens List */}
            <div className="space-y-4">
                {tokens.length === 0 ? (
                    <Card>
                        <CardContent className="flex flex-col items-center justify-center py-12">
                            <div className="text-muted-foreground text-center">
                                <h3 className="font-medium">No API tokens</h3>
                                <p className="text-sm mt-1">Create your first API token to get started</p>
                            </div>
                        </CardContent>
                    </Card>
                ) : (
                    tokens.map((token) => (
                        <Card key={token.id}>
                            <CardContent className="pt-6">
                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <div className="flex items-center space-x-2">
                                            <h3 className="font-medium">{token.name}</h3>
                                            <Badge variant={token.scope === 'write' ? 'default' : 'secondary'}>
                                                {token.scope}
                                            </Badge>
                                            {!token.is_active && (
                                                <Badge variant="destructive">Revoked</Badge>
                                            )}
                                        </div>
                                        {token.description && (
                                            <p className="text-sm text-muted-foreground">{token.description}</p>
                                        )}
                                        <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                                            <span>Prefix: {token.prefix}***</span>
                                            <span>Created: {formatDate(token.created_at)}</span>
                                            {token.last_used_at && (
                                                <span>Last used: {formatDate(token.last_used_at)}</span>
                                            )}
                                        </div>
                                    </div>

                                    <div className="flex items-center space-x-2">
                                        {token.is_active && (
                                            <Button
                                                variant="destructive"
                                                size="sm"
                                                onClick={() => revokeToken(token.id)}
                                            >
                                                <Trash2 className="h-4 w-4 mr-1" />
                                                Revoke
                                            </Button>
                                        )}
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))
                )}
            </div>

            {/* Usage Instructions */}
            <Card>
                <CardHeader>
                    <CardTitle>Using API Tokens</CardTitle>
                    <CardDescription>
                        How to use your API tokens with the Feature Flags platform
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div>
                        <h4 className="font-medium mb-2">Go SDK</h4>
                        <pre className="bg-muted p-3 rounded text-sm overflow-x-auto">
                            {`config := &featureflags.Config{
    APIKey:      "your_api_token_here",
    Environment: "${envId}",
    EvaluatorEndpoint: "http://localhost:8081",
}`}
                        </pre>
                    </div>

                    <div>
                        <h4 className="font-medium mb-2">HTTP API</h4>
                        <pre className="bg-muted p-3 rounded text-sm overflow-x-auto">
                            {`curl -X POST http://localhost:8081/v1/evaluate \\
  -H "Authorization: Bearer your_api_token_here" \\
  -H "Content-Type: application/json" \\
  -d '{
    "env_key": "${envId}",
    "context": {
      "user_key": "user-123",
      "attributes": {"plan": "premium"}
    }
  }'`}
                        </pre>
                    </div>

                    <div className="text-sm text-muted-foreground">
                        <p><strong>Read tokens</strong> can only evaluate flags and retrieve configurations.</p>
                        <p><strong>Write tokens</strong> can also modify flags and configurations (future feature).</p>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
