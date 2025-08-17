'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import apiClient from '@/lib/api';

export default function LoginPage() {
    const router = useRouter();
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState('');
    const [isRegister, setIsRegister] = useState(false);

    const [formData, setFormData] = useState({
        email: 'iamsiddhantmeshram@gmail.com',
        password: 's1i2d3d4',
        firstName: '',
        lastName: '',
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);
        setError('');

        try {
            if (isRegister) {
                const response = await apiClient.register(formData);
                if (response.error) {
                    setError(response.error);
                    return;
                }
                // Auto-login after registration
                const loginResponse = await apiClient.login({
                    email: formData.email,
                    password: formData.password,
                });
                if (loginResponse.error) {
                    setError(loginResponse.error);
                    return;
                }
            } else {
                const response = await apiClient.login({
                    email: formData.email,
                    password: formData.password,
                });
                if (response.error) {
                    setError(response.error);
                    return;
                }
            }

            console.log('Login successful, redirecting to dashboard...');
            router.push('/');
        } catch (err) {
            console.error('Login error:', err);
            setError('An unexpected error occurred');
        } finally {
            setIsLoading(false);
        }
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setFormData(prev => ({
            ...prev,
            [e.target.name]: e.target.value,
        }));
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="space-y-1">
                    <CardTitle className="text-2xl font-bold text-red-400">
                        {isRegister ? 'Create Account' : 'Sign In'}
                    </CardTitle>
                    <CardDescription>
                        {isRegister
                            ? 'Create your account to access the Feature Flag platform'
                            : 'Enter your credentials to access your account'
                        }
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <form onSubmit={handleSubmit} className="space-y-4">
                        {isRegister && (
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <Label htmlFor="firstName">First Name</Label>
                                    <Input
                                        id="firstName"
                                        name="firstName"
                                        type="text"
                                        required={isRegister}
                                        value={formData.firstName}
                                        onChange={handleInputChange}
                                        placeholder="John"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label htmlFor="lastName">Last Name</Label>
                                    <Input
                                        id="lastName"
                                        name="lastName"
                                        type="text"
                                        required={isRegister}
                                        value={formData.lastName}
                                        onChange={handleInputChange}
                                        placeholder="Doe"
                                    />
                                </div>
                            </div>
                        )}

                        <div className="space-y-2">
                            <Label htmlFor="email">Email</Label>
                            <Input
                                id="email"
                                name="email"
                                type="email"
                                required
                                value={formData.email}
                                onChange={handleInputChange}
                                placeholder="test@example.com"
                            />
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="password">Password</Label>
                            <Input
                                id="password"
                                name="password"
                                type="password"
                                required
                                value={formData.password}
                                onChange={handleInputChange}
                                placeholder={isRegister ? "Create a strong password" : "password123"}
                            />
                            <div className="text-xs text-muted-foreground">
                                Default: s1i2d3d4 (your working password)
                            </div>
                        </div>

                        {error && (
                            <Alert variant="destructive">
                                <AlertDescription>{error}</AlertDescription>
                            </Alert>
                        )}

                        <Button
                            type="submit"
                            disabled={isLoading}
                            className="w-full"
                        >
                            {isLoading ? 'Loading...' : isRegister ? 'Create Account' : 'Sign In'}
                        </Button>
                    </form>

                    {!isRegister && (
                        <div className="text-xs text-muted-foreground space-y-1 p-2 bg-muted/50 rounded">
                            <div className="font-medium">Quick password test:</div>
                            <div className="flex gap-2">
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setFormData(prev => ({ ...prev, password: 's1i2d3d4' }))}
                                >
                                    s1i2d3d4
                                </Button>
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() => setFormData(prev => ({ ...prev, password: 'password123' }))}
                                >
                                    password123
                                </Button>
                            </div>
                        </div>
                    )}

                    <div className="relative">
                        <div className="absolute inset-0 flex items-center">
                            <Separator className="w-full" />
                        </div>
                        <div className="relative flex justify-center text-xs uppercase">
                            <span className="bg-background px-2 text-muted-foreground">
                                {isRegister ? 'Already have an account?' : "Don't have an account?"}
                            </span>
                        </div>
                    </div>

                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => setIsRegister(!isRegister)}
                        className="w-full"
                    >
                        {isRegister ? 'Sign In Instead' : 'Create Account'}
                    </Button>
                </CardContent>
            </Card>
        </div>
    );
}