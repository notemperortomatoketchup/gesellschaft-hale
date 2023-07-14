"use client";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import React, { useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import axios from "axios";

export default function page() {
  const [keyword, setKeyword] = useState();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState();
  const [result, setResult] = useState();

  const fetchKeyword = async (keyword) => {
    if (!keyword) {
      setError("Please, fill the keyword in.");
      return;
    }

    setResult();
    setError();
    setIsLoading(true);

    await axios
      .post(
        "https://api.gesellschaft.studio:8443/api/keywordmail",
        {
          keyword: keyword,
          pages: 10,
        },
        {
          headers: {
            Authorization:
              "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MTUsInJvbGUiOjAsInVzZXJuYW1lIjoiZnJvbnRlbmQifQ.M979VNgKzFHsFAnqu_9q3DKJVC5bH7DG2nfrkrd5DBY",
          },
        }
      )
      .then((response) => {
        console.log(response);
        setResult(response.data.data);
      })
      .catch((error) => setError(error.response.data.message));

    setIsLoading(false);
  };

  return (
    <div className=" w-full min-h-screen text-white flex justify-center items-center ">
      <div className="lg:w-1/2 mx-auto flex-col px-4">
        <Card>
          <CardHeader>
            <CardTitle>Scrape keyword</CardTitle>
            <CardDescription>
              Enter the keyword you wish Hale to scrape mails from.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Textarea
              onChange={(e) => setKeyword(() => e.target.value)}
              placeholder="Keyword to search for.."
            />
            <Button
              onClick={() => fetchKeyword(keyword)}
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
