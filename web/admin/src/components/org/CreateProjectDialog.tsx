'use client';

import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const projectSchema = z.object({
    name: z.string().min(1, 'Project name is required').max(100, 'Name must be 100 characters or less'),
    slug: z.string().min(1, 'Project slug is required').max(50, 'Slug must be 50 characters or less')
        .regex(/^[a-z0-9-]+$/, 'Slug can only contain lowercase letters, numbers, and hyphens'),
    description: z.string().max(500, 'Description must be 500 characters or less').optional(),
});

type ProjectData = z.infer<typeof projectSchema>;

interface CreateProjectDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmit: (data: ProjectData) => Promise<void>;
}

export function CreateProjectDialog({ open, onOpenChange, onSubmit }: CreateProjectDialogProps) {
    const form = useForm<ProjectData>({
        resolver: zodResolver(projectSchema),
        defaultValues: {
            name: '',
            slug: '',
            description: '',
        },
    });

    const watchedName = form.watch('name');

    // Auto-generate slug from name
    useEffect(() => {
        if (watchedName && !form.formState.dirtyFields.slug) {
            const slug = watchedName
                .toLowerCase()
                .replace(/[^a-z0-9\s-]/g, '') // Remove special characters
                .replace(/\s+/g, '-') // Replace spaces with hyphens
                .replace(/-+/g, '-') // Replace multiple hyphens with single
                .replace(/^-|-$/g, ''); // Remove leading/trailing hyphens

            form.setValue('slug', slug, { shouldValidate: true });
        }
    }, [watchedName, form]);

    const handleSubmit = async (data: ProjectData) => {
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
                    <DialogTitle>Create Project</DialogTitle>
                    <DialogDescription>
                        Create a new project to organize your feature flags by application or team.
                    </DialogDescription>
                </DialogHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Project Name *</FormLabel>
                                    <FormControl>
                                        <Input
                                            placeholder="My Awesome Project"
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
                            name="slug"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Project Slug *</FormLabel>
                                    <FormControl>
                                        <Input
                                            placeholder="my-awesome-project"
                                            {...field}
                                            className="h-9 font-mono text-sm"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                    <p className="text-xs text-muted-foreground">
                                        Used in URLs and API calls. Auto-generated from name but can be edited.
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
                                            placeholder="Optional description of your project..."
                                            {...field}
                                            className="min-h-[80px] resize-none"
                                        />
                                    </FormControl>
                                    <FormMessage />
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
                                {form.formState.isSubmitting ? 'Creating...' : 'Create Project'}
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
