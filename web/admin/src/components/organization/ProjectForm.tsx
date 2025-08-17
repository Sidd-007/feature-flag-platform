'use client';

import { Button } from '@/components/ui/button';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { Plus } from 'lucide-react';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const projectSchema = z.object({
    name: z.string().min(1, 'Project name is required'),
    slug: z.string().min(1, 'Project slug is required'),
    description: z.string().optional(),
});

type ProjectData = z.infer<typeof projectSchema>;

interface ProjectFormProps {
    onSubmit: (data: ProjectData) => Promise<void>;
    isSubmitting?: boolean;
}

export function ProjectForm({ onSubmit, isSubmitting = false }: ProjectFormProps) {
    const form = useForm<ProjectData>({
        resolver: zodResolver(projectSchema),
        defaultValues: { name: '', slug: '', description: '' },
    });

    // Auto-generate slug from name
    useEffect(() => {
        const subscription = form.watch((value, { name }) => {
            if (name === 'name' && value.name) {
                const slug = value.name
                    .toLowerCase()
                    .replace(/[^a-z0-9\s-]/g, '')
                    .replace(/\s+/g, '-')
                    .replace(/-+/g, '-')
                    .replace(/^-|-$/g, '');
                form.setValue('slug', slug);
            }
        });
        return () => subscription.unsubscribe();
    }, [form]);

    const handleSubmit = async (data: ProjectData) => {
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
                            <FormLabel>Project Name</FormLabel>
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
                            <FormLabel>Project Slug</FormLabel>
                            <FormControl>
                                <Input
                                    placeholder="my-awesome-project"
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
                                    placeholder="Brief description of your project..."
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
                    Create Project
                </Button>
            </form>
        </Form>
    );
}
