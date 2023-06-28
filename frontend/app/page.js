"use client";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import axios from "axios";
import { useEffect, useState } from "react";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle, CheckCircle } from "lucide-react";

export default function Home() {
  const [urls, setUrls] = useState();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState();
  const [result, setResult] = useState();

  const fetchMails = async (urls) => {
    if (!urls) {
      setError("Please, insert at least one website address.");
      return;
    }

    setResult();
    setError();
    setIsLoading(true);
    const splittedUrls = urls.split(" ");
    await axios
      .post("https://api.gesellschaft.studio:8443/api/mail", {
        urls: splittedUrls,
      })
      .then((response) => {
        setResult(response.data.data);
      })
      .catch((error) => setError(error.response.data.message));

    setIsLoading(false);
  };

  return (
    <div className=" w-full h-screen text-white flex justify-center items-center ">
      <div className="w-1/2 mx-auto flex-col">
        <Card>
          <CardHeader>
            <CardTitle>Scrape websites</CardTitle>
            <CardDescription>
              Enter the list of websites to scrape e-mails from.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Textarea
              onChange={(e) => setUrls(() => e.target.value)}
              placeholder="Emails separated by space.."
            />
            <Button
              onClick={() => fetchMails(urls)}
              className="mt-4"
              disabled={isLoading ? true : false}
            >
              {isLoading ? "Scraping..." : "Scrape"}
            </Button>
          </CardContent>
        </Card>
        {error ? (
          <Alert variant="destructive" className="my-4">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {result ? (
          <>
            <Alert className="my-4 border-emerald-800">
              <CheckCircle className="h-4 w-4 !text-emerald-500" />
              <AlertTitle className="text-emerald-500">
                Results scraped!
              </AlertTitle>
              <AlertDescription className="text-emerald-400">
                You can access the results of your scraping task at the bottom
                of the page.
              </AlertDescription>
            </Alert>

            <Table className="mt-24">
              <TableHeader>
                <TableRow>
                  <TableHead>Website</TableHead>
                  <TableHead>Emails</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {result.map((r, index) => (
                  <TableRow key={index}>
                    <TableCell>{r.base_url}</TableCell>
                    <TableCell>
                      {r.mails
                        ? r.mails.map((m, i) => <p key={i}>{m}</p>)
                        : "Not found"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        ) : null}
      </div>
    </div>
  );
}
