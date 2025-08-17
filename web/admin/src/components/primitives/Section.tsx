import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface SectionProps {
    title: string;
    description?: string;
    children: React.ReactNode;
    className?: string;
    headerAction?: React.ReactNode;
}

export function Section({ title, description, children, className, headerAction }: SectionProps) {
    return (
        <Card className={cn("rounded-2xl border bg-card p-5 md:p-6 shadow-sm", className)}>
            <CardHeader className="p-0 pb-5">
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="text-lg font-semibold">{title}</CardTitle>
                        {description && (
                            <CardDescription className="mt-1">{description}</CardDescription>
                        )}
                    </div>
                    {headerAction && (
                        <div>{headerAction}</div>
                    )}
                </div>
            </CardHeader>
            <CardContent className="p-0">
                {children}
            </CardContent>
        </Card>
    );
}
