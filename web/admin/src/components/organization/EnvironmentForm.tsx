'use client';

import { Button } from '@/components/ui/button';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { Plus } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const environmentSchema = z.object({
    name: z.string().min(1, 'Environment name is required'),
    key: z.string().min(1, 'Environment key is required'),
    description: z.string().optional(),
});

type EnvironmentData = z.infer<typeof environmentSchema>;

interface EnvironmentFormProps {
    onSubmit: (data: EnvironmentData) => Promise<void>;
    isSubmitting?: boolean;
}

export function EnvironmentForm({ onSubmit, isSubmitting = false }: EnvironmentFormProps) {
    const form = useForm<EnvironmentData>({
        resolver: zodResolver(environmentSchema),
        defaultValues: { name: '', key: '', description: '' },
    });

    const handleSubmit = async (data: EnvironmentData) => {
        await onSubmit(data);
        form.reset();
    };

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Environment Name</FormLabel>
                            <FormControl>
                                <Input
                                    placeholder="Production"
                                    {...field}
                                    className="h-9"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="key"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Environment Key</FormLabel>
                            <FormControl>
                                <Input
                                    placeholder="prod"
                                    {...field}
                                    className="h-9"
                                />
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
                                    placeholder="Environment description..."
                                    {...field}
                                    className="min-h-[60px]"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button
                    type="submit"
                    className="w-full gap-2"
                    disabled={isSubmitting || form.formState.isSubmitting}
                >
                    <Plus className="h-4 w-4" />
                    Create Environment
                </Button>
            </form>
        </Form>
    );
}
