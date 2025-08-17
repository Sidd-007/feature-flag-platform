'use client';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { zodResolver } from '@hookform/resolvers/zod';
import { Check, ChevronLeft, ChevronRight, Plus, Trash2, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

const flagSchema = z.object({
    key: z.string().min(1, 'Flag key is required').max(100, 'Key must be 100 characters or less')
        .regex(/^[a-z0-9-_]+$/, 'Key can only contain lowercase letters, numbers, hyphens, and underscores'),
    name: z.string().min(1, 'Flag name is required').max(200, 'Name must be 200 characters or less'),
    description: z.string().max(500, 'Description must be 500 characters or less').optional(),
    type: z.enum(['boolean', 'string', 'number', 'json', 'multivariate']),
    enabled: z.boolean(),
    default_value: z.string(),
});

type FlagData = z.infer<typeof flagSchema>;

interface Variant {
    key: string;
    value: string;
    description?: string;
    weight: number;
}

interface Environment {
    id: string;
    name: string;
    key: string;
    description?: string;
    enabled?: boolean;
}

interface EnvironmentOverride {
    envKey: string;
    value: string;
    enabled: boolean;
}

interface CreateFlagStepperProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    environments: Environment[];
    selectedProjectName?: string;
    selectedEnvironmentName?: string;
    onSubmit: (data: FlagData & { variants?: Variant[]; environmentOverrides?: EnvironmentOverride[] }) => Promise<void>;
    initialData?: Partial<FlagData>;
    isEditing?: boolean;
}

const STEPS = [
    { id: 'basics', title: 'Basics', description: 'Key, name, and type' },
    { id: 'variants', title: 'Variants', description: 'For multivariate flags' },
    { id: 'environments', title: 'Environments', description: 'Environment-specific values' },
    { id: 'review', title: 'Review', description: 'Review and create' }
];

export function CreateFlagStepper({
    open,
    onOpenChange,
    environments,
    selectedProjectName,
    selectedEnvironmentName,
    onSubmit,
    initialData,
    isEditing = false
}: CreateFlagStepperProps) {
    const [currentStep, setCurrentStep] = useState(0);
    const [variants, setVariants] = useState<Variant[]>([
        { key: 'control', value: 'false', description: 'Control variant', weight: 50 },
        { key: 'treatment', value: 'true', description: 'Treatment variant', weight: 50 }
    ]);
    const [environmentOverrides, setEnvironmentOverrides] = useState<EnvironmentOverride[]>([]);

    const form = useForm<FlagData>({
        resolver: zodResolver(flagSchema),
        defaultValues: {
            key: initialData?.key || '',
            name: initialData?.name || '',
            description: initialData?.description || '',
            type: initialData?.type || 'boolean',
            enabled: initialData?.enabled ?? true,
            default_value: initialData?.default_value || 'false',
        },
    });

    const watchedName = form.watch('name');
    const watchedType = form.watch('type');
    const watchedDefaultValue = form.watch('default_value');
    const shouldShowVariants = watchedType === 'multivariate';

    // Auto-generate key from name
    useEffect(() => {
        if (watchedName && !form.formState.dirtyFields.key && !initialData?.key) {
            const key = watchedName
                .toLowerCase()
                .replace(/[^a-z0-9\s-_]/g, '')
                .replace(/\s+/g, '_')
                .replace(/[-_]+/g, '_')
                .replace(/^[-_]|[-_]$/g, '');

            form.setValue('key', key, { shouldValidate: true });
        }
    }, [watchedName, form, initialData?.key]);

    // Initialize environment overrides
    useEffect(() => {
        if (environments.length > 0 && environmentOverrides.length === 0) {
            setEnvironmentOverrides(
                environments.map(env => ({
                    envKey: env.key,
                    value: watchedDefaultValue || 'false',
                    enabled: true
                }))
            );
        }
    }, [environments, environmentOverrides.length, watchedDefaultValue]);

    // Keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (!open) return;

            if (e.key === 'Escape') {
                e.preventDefault();
                handleClose();
            } else if (e.key === 'ArrowLeft' && e.ctrlKey) {
                e.preventDefault();
                handlePrevious();
            } else if (e.key === 'ArrowRight' && e.ctrlKey) {
                e.preventDefault();
                if (isStepValid(currentStep)) {
                    handleNext();
                }
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [open, currentStep]);

    const isStepValid = useCallback((stepIndex: number) => {
        switch (stepIndex) {
            case 0: // Basics
                const { key, name, type } = form.getValues();
                return key && name && type;
            case 1: // Variants
                if (!shouldShowVariants) return true;
                const totalWeight = variants.reduce((sum, v) => sum + v.weight, 0);
                return variants.every(v => v.key && v.value) && totalWeight === 100;
            case 2: // Environments
                return environmentOverrides.every(override => override.value);
            case 3: // Review
                return true;
            default:
                return false;
        }
    }, [form, shouldShowVariants, variants, environmentOverrides]);

    const handleNext = () => {
        if (!isStepValid(currentStep)) return;

        if (currentStep === 0 && !shouldShowVariants) {
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

    const handleClose = () => {
        setCurrentStep(0);
        form.reset();
        setVariants([
            { key: 'control', value: 'false', description: 'Control variant', weight: 50 },
            { key: 'treatment', value: 'true', description: 'Treatment variant', weight: 50 }
        ]);
        setEnvironmentOverrides([]);
        onOpenChange(false);
    };

    const handleSubmit = async (data: FlagData) => {
        const submitData = {
            ...data,
            ...(shouldShowVariants && { variants }),
            environmentOverrides
        };

        try {
            await onSubmit(submitData);
            handleClose();
        } catch (error) {
            // Error handling is done in parent component
        }
    };

    // Variant management
    const addVariant = () => {
        const newWeight = Math.max(0, Math.floor((100 - variants.reduce((sum, v) => sum + v.weight, 0)) / 2));
        setVariants([...variants, {
            key: '',
            value: '',
            description: '',
            weight: newWeight
        }]);
    };

    const updateVariant = (index: number, field: keyof Variant, value: string | number) => {
        const newVariants = [...variants];
        (newVariants[index] as any)[field] = value;

        // Redistribute weights if weight changed
        if (field === 'weight') {
            const totalOtherWeights = newVariants.reduce((sum, v, i) =>
                i === index ? sum : sum + v.weight, 0
            );
            const newWeight = value as number;

            if (totalOtherWeights + newWeight <= 100) {
                newVariants[index].weight = newWeight;
            }
        }

        setVariants(newVariants);
    };

    const removeVariant = (index: number) => {
        if (variants.length > 2) {
            setVariants(variants.filter((_, i) => i !== index));
        }
    };

    const setAllToDefault = () => {
        setEnvironmentOverrides(
            environmentOverrides.map(override => ({
                ...override,
                value: watchedDefaultValue,
                enabled: true
            }))
        );
    };

    const updateEnvironmentOverride = (envKey: string, field: keyof EnvironmentOverride, value: string | boolean) => {
        setEnvironmentOverrides(prev =>
            prev.map(override =>
                override.envKey === envKey
                    ? { ...override, [field]: value }
                    : override
            )
        );
    };

    const renderStepIndicator = () => {
        return (
            <nav role="tablist" className="flex items-center justify-center space-x-4 py-6">
                {STEPS.map((step, index) => {
                    // Skip variants step for non-multivariate flags
                    if (index === 1 && !shouldShowVariants) return null;

                    const isActive = index === currentStep;
                    const isCompleted = index < currentStep;
                    const isAvailable = index <= currentStep || isStepValid(index);

                    return (
                        <div key={step.id} className="flex items-center">
                            <div className="flex flex-col items-center">
                                <button
                                    type="button"
                                    role="tab"
                                    aria-current={isActive ? 'step' : undefined}
                                    aria-controls={`step-${index}`}
                                    className={`w-10 h-10 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${isCompleted
                                        ? 'bg-primary text-primary-foreground'
                                        : isActive
                                            ? 'bg-primary text-primary-foreground ring-2 ring-primary ring-offset-2'
                                            : isAvailable
                                                ? 'bg-muted text-muted-foreground hover:bg-muted/80'
                                                : 'bg-muted/50 text-muted-foreground/50'
                                        }`}
                                    onClick={() => isAvailable && setCurrentStep(index)}
                                    disabled={!isAvailable}
                                >
                                    {isCompleted ? (
                                        <Check className="h-5 w-5" />
                                    ) : (
                                        index + 1
                                    )}
                                </button>
                                <div className="text-xs mt-2 text-center max-w-20">
                                    <div className="font-medium">{step.title}</div>
                                </div>
                            </div>

                            {index < STEPS.length - 1 && (shouldShowVariants || index !== 0) && (
                                <div className={`w-12 h-px mx-2 ${isCompleted ? 'bg-primary' : 'bg-border'
                                    }`} />
                            )}
                        </div>
                    );
                })}
            </nav>
        );
    };

    const renderStepContent = () => {
        switch (currentStep) {
            case 0: // Basics
                return (
                    <div id="step-0" className="space-y-4">
                        <FormField
                            control={form.control}
                            name="key"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Flag Key *</FormLabel>
                                    <FormControl>
                                        <Input
                                            placeholder="feature_awesome_new_ui"
                                            {...field}
                                            className="h-9 font-mono"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                    <p className="text-xs text-muted-foreground">
                                        Unique identifier used in code. Auto-generated from name but can be edited.
                                    </p>
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
                                        <Input
                                            placeholder="Awesome New UI"
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
                                    <FormLabel>Description</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            placeholder="What does this flag control?"
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
                            name="type"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Flag Type *</FormLabel>
                                    <FormControl>
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger className="h-9">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="boolean">Boolean - True/False</SelectItem>
                                                <SelectItem value="string">String - Text values</SelectItem>
                                                <SelectItem value="number">Number - Numeric values</SelectItem>
                                                <SelectItem value="json">JSON - Complex objects</SelectItem>
                                                <SelectItem value="multivariate">Multivariate - Multiple variants</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                );

            case 1: // Variants
                if (!shouldShowVariants) return null;

                const totalWeight = variants.reduce((sum, v) => sum + v.weight, 0);
                const remainingWeight = 100 - totalWeight;

                return (
                    <div id="step-1" className="space-y-6">
                        <div>
                            <h4 className="text-lg font-semibold mb-2">Variants</h4>
                            <p className="text-sm text-muted-foreground">
                                Define variants and their traffic allocation. Weights must sum to 100%.
                            </p>

                            {remainingWeight !== 0 && (
                                <div className={`mt-2 text-sm font-medium ${remainingWeight > 0 ? 'text-orange-600' : 'text-red-600'
                                    }`}>
                                    {remainingWeight > 0
                                        ? `${remainingWeight}% unallocated`
                                        : `${Math.abs(remainingWeight)}% over allocated`
                                    }
                                </div>
                            )}
                        </div>

                        <div className="space-y-4">
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
                                                className="gap-1 text-destructive hover:text-destructive"
                                            >
                                                <Trash2 className="h-3 w-3" />
                                                Remove
                                            </Button>
                                        )}
                                    </div>

                                    <div className="grid grid-cols-2 gap-3">
                                        <Input
                                            placeholder="Variant key"
                                            value={variant.key}
                                            onChange={(e) => updateVariant(index, 'key', e.target.value)}
                                            className="h-9"
                                        />
                                        <Input
                                            placeholder="Variant value"
                                            value={variant.value}
                                            onChange={(e) => updateVariant(index, 'value', e.target.value)}
                                            className="h-9"
                                        />
                                    </div>

                                    <Input
                                        placeholder="Description (optional)"
                                        value={variant.description}
                                        onChange={(e) => updateVariant(index, 'description', e.target.value)}
                                        className="h-9"
                                    />

                                    <div className="space-y-2">
                                        <div className="flex items-center justify-between">
                                            <label className="text-sm font-medium">Weight: {variant.weight}%</label>
                                        </div>
                                        <Slider
                                            value={[variant.weight]}
                                            onValueChange={(values: number[]) => updateVariant(index, 'weight', values[0])}
                                            max={100}
                                            step={1}
                                            className="w-full"
                                        />
                                    </div>
                                </div>
                            ))}
                        </div>

                        <Button
                            type="button"
                            variant="outline"
                            onClick={addVariant}
                            className="gap-2"
                        >
                            <Plus className="h-4 w-4" />
                            Add Variant
                        </Button>
                    </div>
                );

            case 2: // Environments
                return (
                    <div id="step-2" className="space-y-6">
                        <div>
                            <h4 className="text-lg font-semibold mb-2">Environment Configuration</h4>
                            <p className="text-sm text-muted-foreground mb-4">
                                Set values for each environment. Default value applies to all environments initially.
                            </p>
                        </div>

                        <FormField
                            control={form.control}
                            name="default_value"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Default Value</FormLabel>
                                    <FormControl>
                                        <Input
                                            placeholder={watchedType === 'boolean' ? 'false' : 'default value'}
                                            {...field}
                                            className="h-9"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                    <div className="flex justify-end">
                                        <Button
                                            type="button"
                                            variant="outline"
                                            size="sm"
                                            onClick={setAllToDefault}
                                            className="text-xs"
                                        >
                                            Set all to default
                                        </Button>
                                    </div>
                                </FormItem>
                            )}
                        />

                        <Separator />

                        <div className="space-y-3">
                            <h5 className="text-sm font-medium">Environment Overrides</h5>
                            {environmentOverrides.map((override) => {
                                const env = environments.find(e => e.key === override.envKey);
                                if (!env) return null;

                                return (
                                    <div key={override.envKey} className="flex items-center gap-3 p-3 border rounded">
                                        <div className="flex-1">
                                            <div className="font-medium text-sm">{env.name}</div>
                                            <div className="text-xs text-muted-foreground">{env.key}</div>
                                        </div>

                                        <Input
                                            value={override.value}
                                            onChange={(e) => updateEnvironmentOverride(override.envKey, 'value', e.target.value)}
                                            className="h-9 w-32"
                                            placeholder="Value"
                                        />

                                        <div className="flex items-center gap-2 min-w-0">
                                            <Switch
                                                checked={override.enabled}
                                                onCheckedChange={(checked) => updateEnvironmentOverride(override.envKey, 'enabled', checked)}
                                            />
                                            <span className="text-xs">
                                                {override.enabled ? 'ON' : 'OFF'}
                                            </span>
                                        </div>

                                        {override.value === watchedDefaultValue ? (
                                            <Badge variant="secondary" className="text-xs">Default</Badge>
                                        ) : (
                                            <Badge variant="outline" className="text-xs">Override</Badge>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    </div>
                );

            case 3: // Review
                return (
                    <div id="step-3" className="space-y-6">
                        <div>
                            <h4 className="text-lg font-semibold mb-2">Review</h4>
                            <p className="text-sm text-muted-foreground">
                                Review your flag configuration before creating.
                            </p>
                        </div>

                        <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <span className="text-sm font-medium">Key:</span>
                                    <code className="text-sm ml-2 bg-muted px-2 py-0.5 rounded font-mono">
                                        {form.watch('key')}
                                    </code>
                                </div>
                                <div>
                                    <span className="text-sm font-medium">Name:</span>
                                    <span className="text-sm ml-2">{form.watch('name')}</span>
                                </div>
                                <div>
                                    <span className="text-sm font-medium">Type:</span>
                                    <Badge variant="outline" className="ml-2">{form.watch('type')}</Badge>
                                </div>
                                <div>
                                    <span className="text-sm font-medium">Default:</span>
                                    <code className="text-sm ml-2 bg-muted px-2 py-0.5 rounded font-mono">
                                        {form.watch('default_value')}
                                    </code>
                                </div>
                            </div>

                            {form.watch('description') && (
                                <div>
                                    <span className="text-sm font-medium">Description:</span>
                                    <p className="text-sm mt-1 text-muted-foreground">{form.watch('description')}</p>
                                </div>
                            )}

                            {shouldShowVariants && variants.length > 0 && (
                                <div>
                                    <span className="text-sm font-medium">Variants:</span>
                                    <div className="mt-2 space-y-1">
                                        {variants.map((variant, index) => (
                                            <div key={index} className="text-sm flex items-center justify-between">
                                                <span>{variant.key}: {variant.value}</span>
                                                <Badge variant="secondary">{variant.weight}%</Badge>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            <div>
                                <span className="text-sm font-medium">Environment Overrides:</span>
                                <div className="mt-2 space-y-1">
                                    {environmentOverrides.map((override) => {
                                        const env = environments.find(e => e.key === override.envKey);
                                        if (!env) return null;

                                        return (
                                            <div key={override.envKey} className="text-sm flex items-center justify-between">
                                                <span>{env.name}:</span>
                                                <div className="flex items-center gap-2">
                                                    <code className="bg-muted px-2 py-0.5 rounded font-mono text-xs">
                                                        {override.value}
                                                    </code>
                                                    <Badge variant={override.enabled ? "default" : "secondary"} className="text-xs">
                                                        {override.enabled ? 'ON' : 'OFF'}
                                                    </Badge>
                                                </div>
                                            </div>
                                        );
                                    })}
                                </div>
                            </div>
                        </div>
                    </div>
                );

            default:
                return null;
        }
    };

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="sm:max-w-3xl max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>
                        {isEditing ? 'Edit' : 'Create'} Feature Flag
                    </DialogTitle>
                    <DialogDescription>
                        {isEditing ? 'Update' : 'Create a new'} feature flag for {selectedProjectName} / {selectedEnvironmentName}
                    </DialogDescription>
                </DialogHeader>

                {/* Step Indicator */}
                {renderStepIndicator()}

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
                        {/* Step Content */}
                        <div className="min-h-[400px]">
                            {renderStepContent()}
                        </div>

                        {/* Footer */}
                        <div className="flex items-center justify-between pt-6 border-t">
                            <Button
                                type="button"
                                variant="outline"
                                onClick={handlePrevious}
                                disabled={currentStep === 0}
                                className="gap-2"
                            >
                                <ChevronLeft className="h-4 w-4" />
                                Previous
                            </Button>

                            <div className="flex gap-2">
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={handleClose}
                                    className="gap-2"
                                >
                                    <X className="h-4 w-4" />
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
                                        className="gap-2"
                                    >
                                        Next
                                        <ChevronRight className="h-4 w-4" />
                                    </Button>
                                )}
                            </div>
                        </div>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
