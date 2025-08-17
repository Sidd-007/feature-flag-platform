"use client";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { Check, Copy } from "lucide-react";
import { useState } from "react";

interface CopyButtonProps {
    text: string;
    className?: string;
    size?: "sm" | "default" | "lg" | "icon";
    variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
}

export function CopyButton({
    text,
    className,
    size = "icon",
    variant = "ghost"
}: CopyButtonProps) {
    const [copied, setCopied] = useState(false);
    const { toast } = useToast();

    const handleCopy = async () => {
        try {
            await navigator.clipboard.writeText(text);
            setCopied(true);
            toast({
                title: "Copied",
                description: "Text copied to clipboard",
            });
            setTimeout(() => setCopied(false), 2000);
        } catch (err) {
            toast({
                title: "Failed to copy",
                description: "Could not copy text to clipboard",
                variant: "destructive",
            });
        }
    };

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Button
                        variant={variant}
                        size={size}
                        onClick={handleCopy}
                        className={cn("h-8 w-8", className)}
                        aria-label={`Copy ${text}`}
                    >
                        {copied ? (
                            <Check className="h-3 w-3" />
                        ) : (
                            <Copy className="h-3 w-3" />
                        )}
                    </Button>
                </TooltipTrigger>
                <TooltipContent>
                    <p>{copied ? "Copied!" : "Copy to clipboard"}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}
