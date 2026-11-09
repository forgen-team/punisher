
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type GoalInput,
  type NoteInput,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

type NoteFilter = { lectureId?: string; conceptId?: string };

export function useNotes(filter: NoteFilter = {}, enabled = true) {
  return useQuery({
    queryKey: punisherKeys.notes(filter),
    queryFn: () => punisherApi.listNotes(filter),
    enabled,
    ...punisherQueryDefaults,
  });
}

export function useCreateNote() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: NoteInput) => punisherApi.createNote(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["punisher", "notes"] }),
  });
}

export function useDeleteNote() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => punisherApi.deleteNote(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["punisher", "notes"] }),
  });
}

export function useGoals() {
  return useQuery({ queryKey: punisherKeys.goals(), queryFn: punisherApi.listGoals, ...punisherQueryDefaults });
}

export function useGoalPath(id: string | undefined) {
  return useQuery({
    queryKey: punisherKeys.goalPath(id ?? ""),
    queryFn: () => punisherApi.getGoalPath(id ?? ""),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
  });
}

function useGoalMutation<TVariables, TResult>(fn: (variables: TVariables) => Promise<TResult>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: punisherKeys.goals() });
      await invalidateLearningState(queryClient);
    },
  });
}

export function useCreateGoal() {
  return useGoalMutation((input: { title: string; targetConceptIds: string[] }) => punisherApi.createGoal(input));
}

export function useUpdateGoal() {
  return useGoalMutation(({ id, input }: { id: string; input: GoalInput }) => punisherApi.updateGoal(id, input));
}

export function useDeleteGoal() {
  return useGoalMutation((id: string) => punisherApi.deleteGoal(id));
}
