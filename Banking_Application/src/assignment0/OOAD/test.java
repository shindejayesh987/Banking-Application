package assignment0.OOAD;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.FileNotFoundException;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;

public class test {

	public static void main(String[] args) 
	{
		String fileName = "/Users/saurabh/Documents/workspace/Assignment_0_OOAD/src/assignment0/OOAD/bank_info.csv";
		String line = null;
		try 
		{
            FileReader fileReader = new FileReader(fileName);
            BufferedReader bufferedReader = new BufferedReader(fileReader);
            BufferedWriter bw = null;
    		FileWriter fw = null;

            while((line = bufferedReader.readLine()) != null) 
            {
            	
            	String[] lines = line.split("	");
            	System.out.println(lines[0]);
            	System.out.println(lines[2]);
            	lines[2] = "3.3";
            	System.out.println(lines[2]);
            }   
            
            bufferedReader.close();         
        }
		catch(FileNotFoundException ex) 
		{
            System.out.println(
                "Unable to open file '" + 
                fileName + "'");                
        }
		catch(IOException ex) 
		{
            System.out.println(
                "Error reading file '" 
                + fileName + "'");                  
            // Or we could just do this: 
            // ex.printStackTrace();
        }
	
	}

}
