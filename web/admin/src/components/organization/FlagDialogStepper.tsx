'use client';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { Check, ChevronLeft, ChevronRight } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const flagSchema = z.object({
    key: z.string().min(1, 'Flag key is required'),
    name: z.string().min(1, 'Flag name is required'),
    description: z.string().optional(),
    type: z.enum(['boolean', 'string', 'number', 'json', 'multivariate']),
    enabled: z.boolean(),
    default_value: z.string(),
    variants: z.array(z.object({
        key: z.string(),
        value: z.string(),
        description: z.string().optional()
    })).optional(),
    environments: z.record(z.string(), z.any()).optional()
});

type FlagData = z.infer<typeof flagSchema>;

interface Environment {
    id: string;
    name: string;
    key: string;
}

interface FlagDialogStepperProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    environments: Environment[];
    selectedProjectName?: string;
    selectedEnvironmentName?: string;
    onSubmit: (data: FlagData) => Promise<void>;
    initialData?: Partial<FlagData>;
    isEditing?: boolean;
}

const STEPS = [
    { id: 'basics', title: 'Basics', description: 'Key, name, and type' },
    { id: 'variants', title: 'Variants', description: 'For multivariate flags' },
    { id: 'environments', title: 'Environments', description: 'Default values per environment' },
    { id: 'review', title: 'Review', description: 'Review and create' }
];

export function FlagDialogStepper({
    open,
    onOpenChange,
    environments,
    selectedProjectName,
    selectedEnvironmentName,
    onSubmit,
    initialData,
    isEditing = false
}: FlagDialogStepperProps) {
    const [currentStep, setCurrentStep] = useState(0);
    const [variants, setVariants] = useState<Array<{ key: string; value: string; description?: string }>>([
        { key: 'control', value: 'false', description: 'Control variant' },
        { key: 'treatment', value: 'true', description: 'Treatment variant' }
    ]);

    const form = useForm<FlagData>({
        resolver: zodResolver(flagSchema),
        defaultValues: {
            key: initialData?.key || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            type: initialData?.type || 'boolean',
            enabled: initialData?.enabled ?? true,
            default_value: initialData?.default_value || 'false',
            ...initialData
        },
    });

    const watchedType = form.watch('type');
    const shouldShowVariants = watchedType === 'multivariate';

    const handleNext = () => {
        if (currentStep === 1 && !shouldShowVariants) {
            setCurrentStep(2); // Skip variants step for non-multivariate flags
        } else if (currentStep < STEPS.length - 1) {
            setCurrentStep(currentStep + 1);
        }
    };

    const handlePrevious = () => {
        if (currentStep === 2 && !shouldShowVariants) {
            setCurrentStep(0); // Skip variants step for non-multivariate flags
        } else if (currentStep > 0) {
            setCurrentStep(currentStep - 1);
        }
    };

    const handleSubmit = async (data: FlagData) => {
        if (shouldShowVariants) {
            data.variants = variants;
        }
        await onSubmit(data);
        onOpenChange(false);
        setCurrentStep(0);
        form.reset();
    };

    const addVariant = () => {
        setVariants([...variants, { key: '', value: '', description: '' }]);
    };

    const updateVariant = (index: number, field: string, value: string) => {
        const newVariants = [...variants];
        (newVariants[index] as any)[field] = value;
        setVariants(newVariants);
    };

    const removeVariant = (index: number) => {
        if (variants.length > 2) {
            setVariants(variants.filter((_, i) => i !== index));
        }
    };

    const isStepValid = (stepIndex: number) => {
        switch (stepIndex) {
            case 0:
                return form.watch('key') && form.watch('name') && form.watch('type');
            case 1:
                return !shouldShowVariants || variants.every(v => v.key && v.value);
            case 2:
                return true;
            case 3:
                return true;
            default:
                return false;
        }
    };

    const renderStepContent = () => {
        switch (currentStep) {
            case 0:
                return (
                    <div className="space-y-4">
                        <FormField
                            control={form.control}
                            name="key"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Flag Key *</FormLabel>
                                    <FormControl>
                                        <Input placeholder="new_feature" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Flag Name *</FormLabel>
                                    <FormControl>
                                        <Input placeholder="New Feature" {...field} />
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
                                    <FormLabel>Description</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            placeholder="Flag description..."
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
                            name="type"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Type *</FormLabel>
                                    <FormControl>
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger>
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="boolean">Boolean</SelectItem>
                                                <SelectItem value="string">String</SelectItem>
                                                <SelectItem value="number">Number</SelectItem>
                                                <SelectItem value="json">JSON</SelectItem>
                                                <SelectItem value="multivariate">Multivariate</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                );

            case 1:
                if (!shouldShowVariants) return null;
                return (
                    <div className="space-y-4">
                        <div>
                            <h4 className="text-sm font-medium mb-2">Variants</h4>
                            <p className="text-sm text-muted-foreground mb-4">
                                Define the different variants for this multivariate flag
                            </p>
                        </div>
                        {variants.map((variant, index) => (
                            <div key={index} className="p-4 border rounded-lg space-y-3">
                                <div className="flex items-center justify-between">
                                    <h5 className="text-sm font-medium">Variant {index + 1}</h5>
                                    {variants.length > 2 && (
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => removeVariant(index)}
                                        >
                                            Remove
                                        </Button>
                                    )}
                                </div>
                                <div className="grid grid-cols-2 gap-3">
                                    <Input
                                        placeholder="Variant key"
                                        value={variant.key}
                                        onChange={(e) => updateVariant(index, 'key', e.target.value)}
                                    />
                                    <Input
                                        placeholder="Variant value"
                                        value={variant.value}
                                        onChange={(e) => updateVariant(index, 'value', e.target.value)}
                                    />
                                </div>
                                <Input
                                    placeholder="Description (optional)"
                                    value={variant.description}
                                    onChange={(e) => updateVariant(index, 'description', e.target.value)}
                                />
                            </div>
                        ))}
                        <Button type="button" variant="outline" onClick={addVariant}>
                            Add Variant
                        </Button>
                    </div>
                );

            case 2:
                return (
                    <div className="space-y-4">
                        <div>
                            <h4 className="text-sm font-medium mb-2">Environment Configuration</h4>
                            <p className="text-sm text-muted-foreground mb-4">
                                Set default values for each environment
                            </p>
                        </div>
                        <FormField
                            control={form.control}
                            name="default_value"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Default Value</FormLabel>
                                    <FormControl>
                                        <Input placeholder="false" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <div className="space-y-3">
                            {environments.map((env) => (
                                <div key={env.id} className="flex items-center justify-between p-3 border rounded">
                                    <div>
                                        <div className="font-medium text-sm">{env.name}</div>
                                        <div className="text-xs text-muted-foreground">{env.key}</div>
                                    </div>
                                    <Badge variant="secondary">Default</Badge>
                                </div>
                            ))}
                        </div>
                    </div>
                );

            case 3:
                return (
                    <div className="space-y-4">
                        <div>
                            <h4 className="text-sm font-medium mb-2">Review</h4>
                            <p className="text-sm text-muted-foreground mb-4">
                                Review your flag configuration before creating
                            </p>
                        </div>
                        <div className="space-y-3 p-4 border rounded-lg bg-muted/20">
                            <div>
                                <span className="text-sm font-medium">Key:</span>
                                <span className="text-sm ml-2">{form.watch('key')}</span>
                            </div>
                            <div>
                                <span className="text-sm font-medium">Name:</span>
                                <span className="text-sm ml-2">{form.watch('name')}</span>
                            </div>
                            <div>
                                <span className="text-sm font-medium">Type:</span>
                                <span className="text-sm ml-2">{form.watch('type')}</span>
                            </div>
                            <div>
                                <span className="text-sm font-medium">Default Value:</span>
                                <span className="text-sm ml-2">{form.watch('default_value')}</span>
                            </div>
                            {shouldShowVariants && variants.length > 0 && (
                                <div>
                                    <span className="text-sm font-medium">Variants:</span>
                                    <div className="mt-1 space-y-1">
                                        {variants.map((variant, index) => (
                                            <div key={index} className="text-sm ml-2">
                                                {variant.key}: {variant.value}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                );

            default:
                return null;
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>
                        {isEditing ? 'Edit' : 'Create'} Feature Flag
                    </DialogTitle>
                    <DialogDescription>
                        {isEditing ? 'Update' : 'Create a new'} feature flag for {selectedProjectName} / {selectedEnvironmentName}
                    </DialogDescription>
                </DialogHeader>

                {/* Steps */}
                <div className="flex items-center justify-center space-x-2 py-4">
                    {STEPS.map((step, index) => {
                        const isActive = index === currentStep;
                        const isCompleted = index < currentStep;
                        const isAvailable = index <= currentStep || isStepValid(index);

                        // Skip variants step for non-multivariate flags
                        if (index === 1 && !shouldShowVariants) return null;

                        return (
                            <div key={step.id} className="flex items-center">
                                <div className="flex flex-col items-center">
                                    <div
                                        className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${isCompleted
                                            ? 'bg-primary text-primary-foreground'
                                            : isActive
                                                ? 'bg-primary text-primary-foreground'
                                                : isAvailable
                                                    ? 'bg-muted text-muted-foreground'
                                                    : 'bg-muted/50 text-muted-foreground/50'
                                            }`}
                                    >
                                        {isCompleted ? (
                                            <Check className="h-4 w-4" />
                                        ) : (
                                            index + 1
                                        )}
                                    </div>
                                    <div className="text-xs mt-1 text-center">
                                        <div className="font-medium">{step.title}</div>
                                    </div>
                                </div>
                                {index < STEPS.length - 1 && (shouldShowVariants || index !== 0) && (
                                    <div className="w-8 h-px bg-border mx-2" />
                                )}
                            </div>
                        );
                    })}
                </div>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                        <div className="min-h-[300px]">
                            {renderStepContent()}
                        </div>

                        <DialogFooter>
                            <div className="flex items-center justify-between w-full">
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={handlePrevious}
                                    disabled={currentStep === 0}
                                    className="gap-1"
                                >
                                    <ChevronLeft className="h-4 w-4" />
                                    Previous
                                </Button>

                                <div className="flex gap-2">
                                    <Button
                                        type="button"
                                        variant="outline"
                                        onClick={() => onOpenChange(false)}
                                    >
                                        Cancel
                                    </Button>

                                    {currentStep === STEPS.length - 1 ? (
                                        <Button
                                            type="submit"
                                            disabled={form.formState.isSubmitting || !isStepValid(currentStep)}
                                        >
                                            {form.formState.isSubmitting
                                                ? (isEditing ? 'Updating...' : 'Creating...')
                                                : (isEditing ? 'Update Flag' : 'Create Flag')
                                            }
                                        </Button>
                                    ) : (
                                        <Button
                                            type="button"
                                            onClick={handleNext}
                                            disabled={!isStepValid(currentStep)}
                                            className="gap-1"
                                        >
                                            Next
                                            <ChevronRight className="h-4 w-4" />
                                        </Button>
                                    )}
                                </div>
                            </div>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
