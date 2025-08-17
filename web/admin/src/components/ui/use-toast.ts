import { toast as sonnerToast } from "sonner"

// Simple toast function that can be used throughout the app
export const toast = ({ title, description, variant }: {
    title: string
    description?: string
    variant?: "default" | "destructive"
}) => {
    if (variant === "destructive") {
        sonnerToast.error(title, {
            description,
        })
    } else {
        sonnerToast.success(title, {
            description,
        })
    }
}

// For backwards compatibility
export { toast as useToast }

