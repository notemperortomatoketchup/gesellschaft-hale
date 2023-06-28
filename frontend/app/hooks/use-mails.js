import { useMutation } from "react-query";

export const useMails = (websites) => {
  const { data, isLoading } = useMutation({
    mutationFn: async () => {
      const { data } = axios.post("https://hale.gesellschaft.studio/mail", {
        websites: websites,
      });
      return data;
    },
  });

  return { data, isLoading };
};
