import { cn } from "@/lib/utils";

type StatusVariant = "success" | "warning" | "error" | "info";

interface StatusDotProps {
    variant: StatusVariant;
    children: React.ReactNode;
    className?: string;
}

const statusVariants = {
    success: "bg-green-500",
    warning: "bg-amber-500",
    error: "bg-red-500",
    info: "bg-blue-500",
};

export function StatusDot({ variant, children, className }: StatusDotProps) {
    return (
        <div className={cn("flex items-center gap-2", className)}>
            <div
                className={cn(
                    "h-2 w-2 rounded-full",
                    statusVariants[variant]
                )}
                aria-hidden="true"
            />
            <span className="text-sm">
                {children}
                <span className="sr-only"> - {variant} status</span>
            </span>
        </div>
    );
}
