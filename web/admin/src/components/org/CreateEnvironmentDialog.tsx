'use client';

import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const environmentSchema = z.object({
    name: z.string().min(1, 'Environment name is required').max(100, 'Name must be 100 characters or less'),
    key: z.string().min(1, 'Environment key is required').max(50, 'Key must be 50 characters or less')
        .regex(/^[a-z0-9-_]+$/, 'Key can only contain lowercase letters, numbers, hyphens, and underscores'),
    description: z.string().max(500, 'Description must be 500 characters or less').optional(),
    enabled: z.boolean(),
});

type EnvironmentData = z.infer<typeof environmentSchema>;

interface CreateEnvironmentDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmit: (data: EnvironmentData) => Promise<void>;
}

export function CreateEnvironmentDialog({ open, onOpenChange, onSubmit }: CreateEnvironmentDialogProps) {
    const form = useForm<EnvironmentData>({
        resolver: zodResolver(environmentSchema),
        defaultValues: {
            name: '',
            key: '',
            description: '',
            enabled: true,
        },
    });

    const watchedName = form.watch('name');

    // Auto-generate key from name
    useEffect(() => {
        if (watchedName && !form.formState.dirtyFields.key) {
            const key = watchedName
                .toLowerCase()
                .replace(/[^a-z0-9\s-_]/g, '') // Remove special characters
                .replace(/\s+/g, '_') // Replace spaces with underscores
                .replace(/[-_]+/g, '_') // Replace multiple hyphens/underscores with single underscore
                .replace(/^[-_]|[-_]$/g, ''); // Remove leading/trailing hyphens/underscores

            form.setValue('key', key, { shouldValidate: true });
        }
    }, [watchedName, form]);

    const handleSubmit = async (data: EnvironmentData) => {
        try {
            await onSubmit(data);
            form.reset();
            onOpenChange(false);
        } catch (error) {
            // Error handling is done in the parent component
        }
    };

    const handleClose = () => {
        form.reset();
        onOpenChange(false);
    };

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Create Environment</DialogTitle>
                    <DialogDescription>
                        Create a new environment to manage feature flags for different stages of your application.
                    </DialogDescription>
                </DialogHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Environment Name *</FormLabel>
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
                                    <FormLabel>Environment Key *</FormLabel>
                                    <FormControl>
                                        <Input
                                            placeholder="production"
                                            {...field}
                                            className="h-9 font-mono text-sm"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                    <p className="text-xs text-muted-foreground">
                                        Used in SDKs and API calls. Auto-generated from name but can be edited.
                                    </p>
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Description</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            placeholder="Optional description of your environment..."
                                            {...field}
                                            className="min-h-[80px] resize-none"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="enabled"
                            render={({ field }) => (
                                <FormItem className="flex items-center justify-between rounded-lg border p-4">
                                    <div className="space-y-0.5">
                                        <FormLabel className="text-base">
                                            Enabled
                                        </FormLabel>
                                        <div className="text-sm text-muted-foreground">
                                            Environment is active and can serve flags
                                        </div>
                                    </div>
                                    <FormControl>
                                        <Switch
                                            checked={field.value}
                                            onCheckedChange={field.onChange}
                                        />
                                    </FormControl>
                                </FormItem>
                            )}
                        />

                        <DialogFooter>
                            <Button type="button" variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button
                                type="submit"
                                disabled={form.formState.isSubmitting || !form.formState.isValid}
                            >
                                {form.formState.isSubmitting ? 'Creating...' : 'Create Environment'}
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
